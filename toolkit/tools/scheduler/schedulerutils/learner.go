// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package schedulerutils

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/pkggraph"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/pkgjson"
)

type RpmIdentity struct {
	FullName   string
	Name       string
	Version    string
	Release    int64
	MarinerVer string
	Arch       string
}

type LearnerResult struct {
	Rpm       RpmIdentity
	BuildTime float32
	Unblocks  []string
}

type Learner struct {
	Results []LearnerResult
}

func NewLearner() (l *Learner) {
	return &Learner{
		Results: make([]LearnerResult, 0),
	}
}

func (l *Learner) RecordUnblocks(dynamicDep *pkgjson.PackageVer, parentNode *pkggraph.PkgNode) {
	logger.Log.Warnf("Dynamic dep info: %#v", dynamicDep)
	logger.Log.Warnf("Provider of dd info: %#v", parentNode)
	var learnerRes = l.GetResult(parentNode.RpmPath)
	learnerRes.Unblocks = append(learnerRes.Unblocks, dynamicDep.Name)
}

func (l *Learner) RecordBuildTime(res *BuildResult) {
	if res.Node.Type == pkggraph.TypeBuild {
		logger.Log.Debugf("address of learner: %p", l)
		var learnerRes = l.GetResult(res.Node.RpmPath)
		logger.Log.Debugf("address of learnerRes: %p", learnerRes)
		logger.Log.Debugf("debuggy: %#v", learnerRes)
		learnerRes.BuildTime = res.BuildTime
		logger.Log.Debugf("debuggy: %f", res.BuildTime)
	}
}

func (l *Learner) Dump(path string) {
	j, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		logger.Log.Error(err)
	}
	file, err := os.Create(path)
	if err != nil {
		logger.Log.Error(err)
	}
	defer file.Close()
	_, err = file.Write(j)
	if err != nil {
		logger.Log.Errorf("Failed to write learner payload, err: %s", err)
	}
	file.Sync()
}

func (l *Learner) GetResult(rpmPath string) (res *LearnerResult) {
	rpmId, err := ParseRpmIdentity(rpmPath)
	if err != nil {
		logger.Log.Warnf("Failed to parse rpm identity for fullRpmPath: %s \n err: %s", rpmPath, err)
	}
	for _, resEntry := range l.Results {
		if resEntry.Rpm.FullName == rpmId.FullName {
			res = &resEntry
			break
		}
	}
	if res == nil {
		res = &LearnerResult{
			Rpm:       rpmId,
			BuildTime: -1,
			Unblocks:  make([]string, 0),
		}
		res.Unblocks = append(res.Unblocks, "foobar")
		l.Results = append(l.Results, *res)
	}
	return
}

func ParseRpmIdentity(fullRpmPath string) (rpmId RpmIdentity, err error) {
	pathParts := strings.Split(fullRpmPath, "/")
	fullName := pathParts[len(pathParts)-1]

	nameParts := strings.Split(fullName, "-")
	name := nameParts[0]
	ver := nameParts[1]
	trailing := nameParts[2]

	trailingParts := strings.Split(trailing, ".")
	rel, err := strconv.ParseInt(trailingParts[0], 10, 64)
	if err != nil {
		return
	}
	marinerVer := trailingParts[1]
	arch := trailingParts[2]

	rpmId = RpmIdentity{
		FullName:   fullName,
		Name:       name,
		Version:    ver,
		Release:    rel,
		MarinerVer: marinerVer,
		Arch:       arch,
	}
	return
}
