package coderd

import (
	"net/http"

	"github.com/google/uuid"
	"nhooyr.io/websocket"

	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/tailnet"
	"github.com/coder/coder/v2/enterprise/wsproxy/wsproxysdk"
)

// @Summary Agent is legacy
// @ID agent-is-legacy
// @Security CoderSessionToken
// @Produce json
// @Tags Enterprise
// @Param workspaceagent path string true "Workspace Agent ID" format(uuid)
// @Success 200 {object} wsproxysdk.AgentIsLegacyResponse
// @Router /workspaceagents/{workspaceagent}/legacy [get]
// @x-apidocgen {"skip": true}
func (api *API) agentIsLegacy(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	agentID, ok := httpmw.ParseUUIDParam(rw, r, "workspaceagent")
	if !ok {
		httpapi.Write(r.Context(), rw, http.StatusBadRequest, codersdk.Response{
			Message: "Missing UUID in URL.",
		})
		return
	}

	node := (*api.AGPL.TailnetCoordinator.Load()).Node(agentID)
	httpapi.Write(ctx, rw, http.StatusOK, wsproxysdk.AgentIsLegacyResponse{
		Found: node != nil,
		Legacy: node != nil &&
			len(node.Addresses) > 0 &&
			node.Addresses[0].Addr() == codersdk.WorkspaceAgentIP,
	})
}

// @Summary Workspace Proxy Coordinate
// @ID workspace-proxy-coordinate
// @Security CoderSessionToken
// @Tags Enterprise
// @Success 101
// @Router /workspaceproxies/me/coordinate [get]
// @x-apidocgen {"skip": true}
func (api *API) workspaceProxyCoordinate(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	api.AGPL.WebsocketWaitMutex.Lock()
	api.AGPL.WebsocketWaitGroup.Add(1)
	api.AGPL.WebsocketWaitMutex.Unlock()
	defer api.AGPL.WebsocketWaitGroup.Done()

	conn, err := websocket.Accept(rw, r, nil)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Failed to accept websocket.",
			Detail:  err.Error(),
		})
		return
	}

	id := uuid.New()
	sub, err := (*api.AGPL.TailnetCoordinator.Load()).ServeMultiAgent(id)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to serve multi agent.",
			Detail:  err.Error(),
		})
		return
	}

	ctx, nc := websocketNetConn(ctx, conn, websocket.MessageText)
	defer nc.Close()

	err = tailnet.ServeWorkspaceProxy(ctx, nc, sub)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, err.Error())
	}
}
