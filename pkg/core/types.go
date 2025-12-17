package core

// Reactor Interface Version 2
type Reactor interface {
	// OnStart nhận FluxorContext thay vì context thường
	OnStart(ctx *FluxorContext) error
	
	// OnStop để dọn dẹp resource
	OnStop() error
}
