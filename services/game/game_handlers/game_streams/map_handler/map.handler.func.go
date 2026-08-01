package map_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_internals/worldmap_conns"
)

// HandlerMapData 查询视野地图数据 (1000007)
//
// 透传 worldmap 的 MapData 查询，返回 AOI 视野内地块 + 行军。
func HandlerMapData(ctx context.Context, _ uint64, req *pb_worldmap.MapDataReq, resp *pb_worldmap.MapDataRsp) rpc_results.ResultI {
	if req.GetMapId() < 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid map_id")
	}

	rsp, err := worldmap_conns.MapData(ctx, req)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("map data failed: %s", err.Error()))
	}

	resp.Cells = rsp.GetCells()
	resp.Marches = rsp.GetMarches()
	return nil
}
