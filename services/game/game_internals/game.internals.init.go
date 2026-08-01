package game_internals

import (
	"context"

	"server.slg.com/services/game/game_internals/game_rpc_clients"
	"server.slg.com/services/game/game_internals/gate_stream"
	"server.slg.com/services/game/game_internals/stream_consumers"
)

func Init(ctx context.Context) {
	gate_stream.Init(ctx)
	stream_consumers.Init(ctx)
	game_rpc_clients.Init(ctx)
}
