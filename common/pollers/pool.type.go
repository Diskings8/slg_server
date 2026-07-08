package pollers

type LoaderFunc[M DataI] func(id uint64) (M, error)
