package common_declarations

type NodeService int

const (
	NodeGameService     NodeService = 10
	NodeGatewayService  NodeService = 20
	NodeWorldMapService NodeService = 30
)

type LoaderFunc[M DataI] func(id uint64) (M, error)
