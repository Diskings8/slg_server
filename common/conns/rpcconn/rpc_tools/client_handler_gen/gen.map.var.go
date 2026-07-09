package main

// 新增 proto service 后，如果 nodeType 映射不对，在这里加一行即可
var pkgNodeMap = map[string]string{
	"pb_game":     "NodeGameService",
	"pb_gateway":  "NodeGatewayService",
	"pb_worldmap": "NodeWorldMapService",
}
