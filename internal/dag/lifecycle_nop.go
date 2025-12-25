package dag

import "context"

// NopLifecycleHooks is a no-op LifecycleHooks implementation.
type NopLifecycleHooks struct{}

func (NopLifecycleHooks) BeforeRun(context.Context)             {}
func (NopLifecycleHooks) AfterRun(context.Context)              {}
func (NopLifecycleHooks) BeforeNode(context.Context, string)    {}
func (NopLifecycleHooks) AfterNode(context.Context, string)     {}
