package role_heroes

import "server.slg.com/services/game/game_models"

type Heroes struct {
	List []*game_models.RoleHero
}

type Hero struct {
	*game_models.RoleHero
}
