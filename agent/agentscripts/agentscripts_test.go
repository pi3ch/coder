package agentscripts_test

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/goleak"

	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/agent/agentscripts"
	"github.com/coder/coder/v2/agent/agentssh"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/codersdk/agentsdk"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestExecuteBasic(t *testing.T) {
	t.Parallel()
	logs := make(chan agentsdk.PatchLogs, 1)
	runner := setup(t, func(ctx context.Context, req agentsdk.PatchLogs) error {
		logs <- req
		return nil
	})
	defer runner.Close()
	err := runner.Init([]codersdk.WorkspaceAgentScript{{
		LogSourceDisplayName: "test",
		Source:               "echo hello",
	}})
	require.NoError(t, err)
	require.NoError(t, runner.Execute(func(script codersdk.WorkspaceAgentScript) bool {
		return true
	}))
	log := <-logs
	require.Equal(t, "hello", log.Logs[0].Output)
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	runner := setup(t, nil)
	defer runner.Close()
	err := runner.Init([]codersdk.WorkspaceAgentScript{{
		Source:  "sleep 3",
		Timeout: time.Nanosecond,
	}})
	require.NoError(t, err)
	require.ErrorIs(t, runner.Execute(nil), agentscripts.ErrTimeout)
}

func setup(t *testing.T, patchLogs func(ctx context.Context, req agentsdk.PatchLogs) error) *agentscripts.Runner {
	t.Helper()
	if patchLogs == nil {
		// noop
		patchLogs = func(ctx context.Context, req agentsdk.PatchLogs) error {
			return nil
		}
	}
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := slogtest.Make(t, nil)
	s, err := agentssh.NewServer(ctx, logger, prometheus.NewRegistry(), fs, 0, "")
	require.NoError(t, err)
	s.AgentToken = func() string { return "" }
	s.Manifest = atomic.NewPointer(&agentsdk.Manifest{})
	t.Cleanup(func() {
		_ = s.Close()
	})
	return agentscripts.New(ctx, agentscripts.Options{
		LogDir:     t.TempDir(),
		Logger:     logger,
		SSHServer:  s,
		Filesystem: fs,
		PatchLogs:  patchLogs,
	})
}