// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s RepositoryService) ListCommits(request *rpc.ListCommitsRequest,
	stream rpc.RepositoryService_ListCommitsServer) error {
	repoPath := s.getFullPathForRepo(request.GetRepoUid())

	gitCommits, totalCount, err := s.adapter.ListCommits(stream.Context(), repoPath, request.GetGitRef(),
		int(request.GetPage()), int(request.GetPageSize()))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list commits: %v", err)
	}

	log.Trace().Msgf("git adapter returned %d commits (total: %d)", len(gitCommits), totalCount)

	// send info about total number of commits first
	err = stream.Send(&rpc.ListCommitsResponse{
		Data: &rpc.ListCommitsResponse_Header{
			Header: &rpc.ListCommitsResponseHeader{
				TotalCount: totalCount,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send response header: %v", err)
	}

	for i := range gitCommits {
		var commit *rpc.Commit
		commit, err = mapGitCommit(&gitCommits[i])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
		}

		err = stream.Send(&rpc.ListCommitsResponse{
			Data: &rpc.ListCommitsResponse_Commit{
				Commit: commit,
			},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send commit: %v", err)
		}
	}

	return nil
}

func (s RepositoryService) getLatestCommit(ctx context.Context, repoPath string,
	ref string, path string) (*rpc.Commit, error) {
	gitCommit, err := s.adapter.GetLatestCommit(ctx, repoPath, ref, path)
	if err != nil {
		return nil, err
	}

	return mapGitCommit(gitCommit)
}
