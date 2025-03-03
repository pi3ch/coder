//go:build linux

package database_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbgen"
	"github.com/coder/coder/v2/coderd/database/dbtime"
	"github.com/coder/coder/v2/coderd/database/migrations"
	"github.com/coder/coder/v2/testutil"
)

func TestGetDeploymentWorkspaceAgentStats(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.SkipNow()
	}
	t.Run("Aggregates", func(t *testing.T) {
		t.Parallel()
		sqlDB := testSQLDB(t)
		err := migrations.Up(sqlDB)
		require.NoError(t, err)
		db := database.New(sqlDB)
		ctx := context.Background()
		dbgen.WorkspaceAgentStat(t, db, database.WorkspaceAgentStat{
			TxBytes:                   1,
			RxBytes:                   1,
			ConnectionMedianLatencyMS: 1,
			SessionCountVSCode:        1,
		})
		dbgen.WorkspaceAgentStat(t, db, database.WorkspaceAgentStat{
			TxBytes:                   1,
			RxBytes:                   1,
			ConnectionMedianLatencyMS: 2,
			SessionCountVSCode:        1,
		})
		stats, err := db.GetDeploymentWorkspaceAgentStats(ctx, dbtime.Now().Add(-time.Hour))
		require.NoError(t, err)

		require.Equal(t, int64(2), stats.WorkspaceTxBytes)
		require.Equal(t, int64(2), stats.WorkspaceRxBytes)
		require.Equal(t, 1.5, stats.WorkspaceConnectionLatency50)
		require.Equal(t, 1.95, stats.WorkspaceConnectionLatency95)
		require.Equal(t, int64(2), stats.SessionCountVSCode)
	})

	t.Run("GroupsByAgentID", func(t *testing.T) {
		t.Parallel()

		sqlDB := testSQLDB(t)
		err := migrations.Up(sqlDB)
		require.NoError(t, err)
		db := database.New(sqlDB)
		ctx := context.Background()
		agentID := uuid.New()
		insertTime := dbtime.Now()
		dbgen.WorkspaceAgentStat(t, db, database.WorkspaceAgentStat{
			CreatedAt:                 insertTime.Add(-time.Second),
			AgentID:                   agentID,
			TxBytes:                   1,
			RxBytes:                   1,
			ConnectionMedianLatencyMS: 1,
			SessionCountVSCode:        1,
		})
		dbgen.WorkspaceAgentStat(t, db, database.WorkspaceAgentStat{
			// Ensure this stat is newer!
			CreatedAt:                 insertTime,
			AgentID:                   agentID,
			TxBytes:                   1,
			RxBytes:                   1,
			ConnectionMedianLatencyMS: 2,
			SessionCountVSCode:        1,
		})
		stats, err := db.GetDeploymentWorkspaceAgentStats(ctx, dbtime.Now().Add(-time.Hour))
		require.NoError(t, err)

		require.Equal(t, int64(2), stats.WorkspaceTxBytes)
		require.Equal(t, int64(2), stats.WorkspaceRxBytes)
		require.Equal(t, 1.5, stats.WorkspaceConnectionLatency50)
		require.Equal(t, 1.95, stats.WorkspaceConnectionLatency95)
		require.Equal(t, int64(1), stats.SessionCountVSCode)
	})
}

func TestInsertWorkspaceAgentLogs(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.SkipNow()
	}
	sqlDB := testSQLDB(t)
	ctx := context.Background()
	err := migrations.Up(sqlDB)
	require.NoError(t, err)
	db := database.New(sqlDB)
	org := dbgen.Organization(t, db, database.Organization{})
	job := dbgen.ProvisionerJob(t, db, nil, database.ProvisionerJob{
		OrganizationID: org.ID,
	})
	resource := dbgen.WorkspaceResource(t, db, database.WorkspaceResource{
		JobID: job.ID,
	})
	agent := dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{
		ResourceID: resource.ID,
	})
	logs, err := db.InsertWorkspaceAgentLogs(ctx, database.InsertWorkspaceAgentLogsParams{
		AgentID:   agent.ID,
		CreatedAt: []time.Time{dbtime.Now()},
		Output:    []string{"first"},
		Level:     []database.LogLevel{database.LogLevelInfo},
		Source:    []database.WorkspaceAgentLogSource{database.WorkspaceAgentLogSourceExternal},
		// 1 MB is the max
		OutputLength: 1 << 20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), logs[0].ID)

	_, err = db.InsertWorkspaceAgentLogs(ctx, database.InsertWorkspaceAgentLogsParams{
		AgentID:      agent.ID,
		CreatedAt:    []time.Time{dbtime.Now()},
		Output:       []string{"second"},
		Level:        []database.LogLevel{database.LogLevelInfo},
		Source:       []database.WorkspaceAgentLogSource{database.WorkspaceAgentLogSourceExternal},
		OutputLength: 1,
	})
	require.True(t, database.IsWorkspaceAgentLogsLimitError(err))
}

func TestProxyByHostname(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.SkipNow()
	}
	sqlDB := testSQLDB(t)
	err := migrations.Up(sqlDB)
	require.NoError(t, err)
	db := database.New(sqlDB)

	// Insert a bunch of different proxies.
	proxies := []struct {
		name             string
		accessURL        string
		wildcardHostname string
	}{
		{
			name:             "one",
			accessURL:        "https://one.coder.com",
			wildcardHostname: "*.wildcard.one.coder.com",
		},
		{
			name:             "two",
			accessURL:        "https://two.coder.com",
			wildcardHostname: "*--suffix.two.coder.com",
		},
	}
	for _, p := range proxies {
		dbgen.WorkspaceProxy(t, db, database.WorkspaceProxy{
			Name:             p.name,
			Url:              p.accessURL,
			WildcardHostname: p.wildcardHostname,
		})
	}

	cases := []struct {
		name              string
		testHostname      string
		allowAccessURL    bool
		allowWildcardHost bool
		matchProxyName    string
	}{
		{
			name:              "NoMatch",
			testHostname:      "test.com",
			allowAccessURL:    true,
			allowWildcardHost: true,
			matchProxyName:    "",
		},
		{
			name:              "MatchAccessURL",
			testHostname:      "one.coder.com",
			allowAccessURL:    true,
			allowWildcardHost: true,
			matchProxyName:    "one",
		},
		{
			name:              "MatchWildcard",
			testHostname:      "something.wildcard.one.coder.com",
			allowAccessURL:    true,
			allowWildcardHost: true,
			matchProxyName:    "one",
		},
		{
			name:              "MatchSuffix",
			testHostname:      "something--suffix.two.coder.com",
			allowAccessURL:    true,
			allowWildcardHost: true,
			matchProxyName:    "two",
		},
		{
			name:              "ValidateHostname/1",
			testHostname:      ".*ne.coder.com",
			allowAccessURL:    true,
			allowWildcardHost: true,
			matchProxyName:    "",
		},
		{
			name:              "ValidateHostname/2",
			testHostname:      "https://one.coder.com",
			allowAccessURL:    true,
			allowWildcardHost: true,
			matchProxyName:    "",
		},
		{
			name:              "ValidateHostname/3",
			testHostname:      "one.coder.com:8080/hello",
			allowAccessURL:    true,
			allowWildcardHost: true,
			matchProxyName:    "",
		},
		{
			name:              "IgnoreAccessURLMatch",
			testHostname:      "one.coder.com",
			allowAccessURL:    false,
			allowWildcardHost: true,
			matchProxyName:    "",
		},
		{
			name:              "IgnoreWildcardMatch",
			testHostname:      "hi.wildcard.one.coder.com",
			allowAccessURL:    true,
			allowWildcardHost: false,
			matchProxyName:    "",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			proxy, err := db.GetWorkspaceProxyByHostname(context.Background(), database.GetWorkspaceProxyByHostnameParams{
				Hostname:              c.testHostname,
				AllowAccessUrl:        c.allowAccessURL,
				AllowWildcardHostname: c.allowWildcardHost,
			})
			if c.matchProxyName == "" {
				require.ErrorIs(t, err, sql.ErrNoRows)
				require.Empty(t, proxy)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, proxy)
				require.Equal(t, c.matchProxyName, proxy.Name)
			}
		})
	}
}

func TestDefaultProxy(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.SkipNow()
	}
	sqlDB := testSQLDB(t)
	err := migrations.Up(sqlDB)
	require.NoError(t, err)
	db := database.New(sqlDB)

	ctx := testutil.Context(t, testutil.WaitLong)
	depID := uuid.NewString()
	err = db.InsertDeploymentID(ctx, depID)
	require.NoError(t, err, "insert deployment id")

	// Fetch empty proxy values
	defProxy, err := db.GetDefaultProxyConfig(ctx)
	require.NoError(t, err, "get def proxy")

	require.Equal(t, defProxy.DisplayName, "Default")
	require.Equal(t, defProxy.IconUrl, "/emojis/1f3e1.png")

	// Set the proxy values
	args := database.UpsertDefaultProxyParams{
		DisplayName: "displayname",
		IconUrl:     "/icon.png",
	}
	err = db.UpsertDefaultProxy(ctx, args)
	require.NoError(t, err, "insert def proxy")

	defProxy, err = db.GetDefaultProxyConfig(ctx)
	require.NoError(t, err, "get def proxy")
	require.Equal(t, defProxy.DisplayName, args.DisplayName)
	require.Equal(t, defProxy.IconUrl, args.IconUrl)

	// Upsert values
	args = database.UpsertDefaultProxyParams{
		DisplayName: "newdisplayname",
		IconUrl:     "/newicon.png",
	}
	err = db.UpsertDefaultProxy(ctx, args)
	require.NoError(t, err, "upsert def proxy")

	defProxy, err = db.GetDefaultProxyConfig(ctx)
	require.NoError(t, err, "get def proxy")
	require.Equal(t, defProxy.DisplayName, args.DisplayName)
	require.Equal(t, defProxy.IconUrl, args.IconUrl)

	// Ensure other site configs are the same
	found, err := db.GetDeploymentID(ctx)
	require.NoError(t, err, "get deployment id")
	require.Equal(t, depID, found)
}

func TestQueuePosition(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.SkipNow()
	}
	sqlDB := testSQLDB(t)
	err := migrations.Up(sqlDB)
	require.NoError(t, err)
	db := database.New(sqlDB)
	ctx := testutil.Context(t, testutil.WaitLong)

	org := dbgen.Organization(t, db, database.Organization{})
	jobCount := 10
	jobs := []database.ProvisionerJob{}
	jobIDs := []uuid.UUID{}
	for i := 0; i < jobCount; i++ {
		job := dbgen.ProvisionerJob(t, db, nil, database.ProvisionerJob{
			OrganizationID: org.ID,
			Tags:           database.StringMap{},
		})
		jobs = append(jobs, job)
		jobIDs = append(jobIDs, job.ID)

		// We need a slight amount of time between each insertion to ensure that
		// the queue position is correct... it's sorted by `created_at`.
		time.Sleep(time.Millisecond)
	}

	queued, err := db.GetProvisionerJobsByIDsWithQueuePosition(ctx, jobIDs)
	require.NoError(t, err)
	require.Len(t, queued, jobCount)
	sort.Slice(queued, func(i, j int) bool {
		return queued[i].QueuePosition < queued[j].QueuePosition
	})
	// Ensure that the queue positions are correct based on insertion ID!
	for index, job := range queued {
		require.Equal(t, job.QueuePosition, int64(index+1))
		require.Equal(t, job.ProvisionerJob.ID, jobs[index].ID)
	}

	job, err := db.AcquireProvisionerJob(ctx, database.AcquireProvisionerJobParams{
		StartedAt: sql.NullTime{
			Time:  dbtime.Now(),
			Valid: true,
		},
		Types: database.AllProvisionerTypeValues(),
		WorkerID: uuid.NullUUID{
			UUID:  uuid.New(),
			Valid: true,
		},
		Tags: json.RawMessage("{}"),
	})
	require.NoError(t, err)
	require.Equal(t, jobs[0].ID, job.ID)

	queued, err = db.GetProvisionerJobsByIDsWithQueuePosition(ctx, jobIDs)
	require.NoError(t, err)
	require.Len(t, queued, jobCount)
	sort.Slice(queued, func(i, j int) bool {
		return queued[i].QueuePosition < queued[j].QueuePosition
	})
	// Ensure that queue positions are updated now that the first job has been acquired!
	for index, job := range queued {
		if index == 0 {
			require.Equal(t, job.QueuePosition, int64(0))
			continue
		}
		require.Equal(t, job.QueuePosition, int64(index))
		require.Equal(t, job.ProvisionerJob.ID, jobs[index].ID)
	}
}

func TestUserLastSeenFilter(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.SkipNow()
	}
	t.Run("Before", func(t *testing.T) {
		t.Parallel()
		sqlDB := testSQLDB(t)
		err := migrations.Up(sqlDB)
		require.NoError(t, err)
		db := database.New(sqlDB)
		ctx := context.Background()
		now := dbtime.Now()

		yesterday := dbgen.User(t, db, database.User{
			LastSeenAt: now.Add(time.Hour * -25),
		})
		today := dbgen.User(t, db, database.User{
			LastSeenAt: now,
		})
		lastWeek := dbgen.User(t, db, database.User{
			LastSeenAt: now.Add((time.Hour * -24 * 7) + (-1 * time.Hour)),
		})

		beforeToday, err := db.GetUsers(ctx, database.GetUsersParams{
			LastSeenBefore: now.Add(time.Hour * -24),
		})
		require.NoError(t, err)
		database.ConvertUserRows(beforeToday)

		requireUsersMatch(t, []database.User{yesterday, lastWeek}, beforeToday, "before today")

		justYesterday, err := db.GetUsers(ctx, database.GetUsersParams{
			LastSeenBefore: now.Add(time.Hour * -24),
			LastSeenAfter:  now.Add(time.Hour * -24 * 2),
		})
		require.NoError(t, err)
		requireUsersMatch(t, []database.User{yesterday}, justYesterday, "just yesterday")

		all, err := db.GetUsers(ctx, database.GetUsersParams{
			LastSeenBefore: now.Add(time.Hour),
		})
		require.NoError(t, err)
		requireUsersMatch(t, []database.User{today, yesterday, lastWeek}, all, "all")

		allAfterLastWeek, err := db.GetUsers(ctx, database.GetUsersParams{
			LastSeenAfter: now.Add(time.Hour * -24 * 7),
		})
		require.NoError(t, err)
		requireUsersMatch(t, []database.User{today, yesterday}, allAfterLastWeek, "after last week")
	})
}

func TestUserChangeLoginType(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.SkipNow()
	}

	sqlDB := testSQLDB(t)
	err := migrations.Up(sqlDB)
	require.NoError(t, err)
	db := database.New(sqlDB)
	ctx := context.Background()

	alice := dbgen.User(t, db, database.User{
		LoginType: database.LoginTypePassword,
	})
	bob := dbgen.User(t, db, database.User{
		LoginType: database.LoginTypePassword,
	})
	bobExpPass := bob.HashedPassword
	require.NotEmpty(t, alice.HashedPassword, "hashed password should not start empty")
	require.NotEmpty(t, bob.HashedPassword, "hashed password should not start empty")

	alice, err = db.UpdateUserLoginType(ctx, database.UpdateUserLoginTypeParams{
		NewLoginType: database.LoginTypeOIDC,
		UserID:       alice.ID,
	})
	require.NoError(t, err)

	require.Empty(t, alice.HashedPassword, "hashed password should be empty")

	// First check other users are not affected
	bob, err = db.GetUserByID(ctx, bob.ID)
	require.NoError(t, err)
	require.Equal(t, bobExpPass, bob.HashedPassword, "hashed password should not change")

	// Then check password -> password is a noop
	bob, err = db.UpdateUserLoginType(ctx, database.UpdateUserLoginTypeParams{
		NewLoginType: database.LoginTypePassword,
		UserID:       bob.ID,
	})
	require.NoError(t, err)

	bob, err = db.GetUserByID(ctx, bob.ID)
	require.NoError(t, err)
	require.Equal(t, bobExpPass, bob.HashedPassword, "hashed password should not change")
}

func requireUsersMatch(t testing.TB, expected []database.User, found []database.GetUsersRow, msg string) {
	t.Helper()
	require.ElementsMatch(t, expected, database.ConvertUserRows(found), msg)
}
