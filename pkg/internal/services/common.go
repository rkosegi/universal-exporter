package services

import "github.com/rkosegi/yaml-toolkit/pipeline"

type noopService struct{}

func (n noopService) Configure(pipeline.ServiceContext, pipeline.StrKeysAnyValues) pipeline.Service {
	return nil
}
func (n noopService) Init() error  { return nil }
func (n noopService) Close() error { return nil }
