package config

import (
	"context"
	"sort"
	"spoutmc/internal/docker"
	"spoutmc/internal/log"
	"spoutmc/internal/models"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
)

var logger = log.GetLogger(log.ModuleConfig)

type ServerChange struct {
	Key    string
	Before models.SpoutServer
	After  models.SpoutServer
	Diff   string
}

type ChangeSet struct {
	Added   []models.SpoutServer // present in new, not in old
	Removed []models.SpoutServer // present in old, not in new
	Updated []ServerChange       // present in both but different
}

func DiffServers(oldList, newList []models.SpoutServer) ChangeSet {
	keyFn := func(s models.SpoutServer) string { return s.Name }

	oldMap := indexByKey(oldList, keyFn)
	newMap := indexByKey(newList, keyFn)

	var cs ChangeSet

	for k, ov := range oldMap {
		nv, stillThere := newMap[k]
		if !stillThere {
			cs.Removed = append(cs.Removed, ov)
			continue
		}
		if !cmp.Equal(ov, nv, cmpOptions()...) {
			cs.Updated = append(cs.Updated, ServerChange{
				Key:    k,
				Before: ov,
				After:  nv,
				Diff:   cmp.Diff(ov, nv, cmpOptions()...),
			})
		}
	}

	for k, nv := range newMap {
		if _, had := oldMap[k]; !had {
			cs.Added = append(cs.Added, nv)
		}
	}

	sort.Slice(cs.Added, func(i, j int) bool { return keyFn(cs.Added[i]) < keyFn(cs.Added[j]) })
	sort.Slice(cs.Removed, func(i, j int) bool { return keyFn(cs.Removed[i]) < keyFn(cs.Removed[j]) })
	sort.Slice(cs.Updated, func(i, j int) bool { return cs.Updated[i].Key < cs.Updated[j].Key })

	return cs
}

func indexByKey(list []models.SpoutServer, keyFn func(models.SpoutServer) string) map[string]models.SpoutServer {
	m := make(map[string]models.SpoutServer, len(list))
	for _, s := range list {
		m[keyFn(s)] = s
	}
	return m
}

func cmpOptions() []cmp.Option {
	return []cmp.Option{
		cmpopts.EquateEmpty(),
	}
}

func ApplyConfigChanges(ctx context.Context, oldConfig, newConfig models.SpoutConfiguration) {

	changeSet := DiffServers(oldConfig.Servers, newConfig.Servers)

	dataPath := ""
	if newConfig.Storage != nil {
		dataPath = newConfig.Storage.DataPath
	}

	if len(changeSet.Updated) > 0 {
		for _, changed := range changeSet.Updated {
			err := docker.RecreateContainer(ctx, changed.After, dataPath)
			if err != nil {
				logger.Error("cannot recreate container", zap.Error(err))
			}

		}
	}

	if len(changeSet.Added) > 0 {
		for _, added := range changeSet.Added {
			if err := docker.StartContainer(ctx, added, dataPath); err != nil {
				logger.Error("failed to start added server",
					zap.String("server", added.Name),
					zap.Error(err))
			}
		}
	}
	if len(changeSet.Removed) > 0 {
		for _, removed := range changeSet.Removed {
			removedContainer, _ := docker.GetContainer(ctx, removed.Name)

			err := docker.StopAndRemoveContainerById(ctx, removedContainer.ID)
			if err != nil {
				logger.Error("cannot remove container", zap.Error(err))
			}
		}
	}

	hasServerChanges := len(changeSet.Added) > 0 || len(changeSet.Updated) > 0 || len(changeSet.Removed) > 0
	if hasServerChanges {
		if err := docker.SyncVelocityTomlAndRestartProxy(ctx, &newConfig); err != nil {
			logger.Error("failed to sync velocity.toml and restart proxy after config changes", zap.Error(err))
		}
	}

}
