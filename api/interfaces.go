package api

type SubSystem interface {
	End()
}

type Updatable interface {
	SubSystem
	Update(dt float64)
}
