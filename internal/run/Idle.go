package run

import (
	"time"

	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/context"
)

type Idle struct {
	ctx *context.Status
}

func NewIdle() *Idle {
	return &Idle{
		ctx: context.Get(),
	}
}

func (a Idle) Name() string {
	return string(config.IdleRun)
}

func (a Idle) Run() error {
	idleTimeSeconds := a.ctx.CharacterCfg.Game.Idle.Idletime
	a.ctx.Logger.Info("Starting idle period", "duration", idleTimeSeconds)

	idleDuration := time.Duration(idleTimeSeconds) * time.Second
	time.Sleep(idleDuration)

	return nil
}
