// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protection

import (
	"context"
	"reflect"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// nolint:gocognit // it's a unit test
func TestRuleSet_MergeVerify(t *testing.T) {
	tests := []struct {
		name    string
		rules   []types.RuleInfoInternal
		input   MergeVerifyInput
		expOut  MergeVerifyOutput
		expViol []types.RuleViolations
	}{
		{
			name:  "empty-with-merge-method",
			rules: []types.RuleInfoInternal{},
			input: MergeVerifyInput{
				Actor:      &types.Principal{ID: 1},
				Method:     enum.MergeMethodRebase,
				TargetRepo: &types.Repository{ID: 1, DefaultBranch: "main"},
				PullReq:    &types.PullReq{ID: 1, SourceBranch: "pr", TargetBranch: "main"},
			},
			expOut: MergeVerifyOutput{
				DeleteSourceBranch: false,
				AllowedMethods:     nil,
			},
			expViol: nil,
		},
		{
			name:  "empty-no-merge-method-specified",
			rules: []types.RuleInfoInternal{},
			input: MergeVerifyInput{
				Actor:      &types.Principal{ID: 1},
				TargetRepo: &types.Repository{ID: 1, DefaultBranch: "main"},
				PullReq:    &types.PullReq{ID: 1, SourceBranch: "pr", TargetBranch: "main"},
			},
			expOut: MergeVerifyOutput{
				DeleteSourceBranch: false,
				AllowedMethods:     enum.MergeMethods,
			},
			expViol: nil,
		},
		{
			name: "two-rules-delete-source-branch",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo: types.RuleInfo{
						SpacePath:  "",
						RepoPath:   "space/repo",
						ID:         1,
						Identifier: "rule1",
						Type:       TypeBranch,
						State:      enum.RuleStateActive,
					},
					Pattern:    []byte(`{"default":true}`),
					Definition: []byte(`{"pullreq":{"merge":{"strategies_allowed":["merge"],"delete_branch":true}}}`),
				},
				{
					RuleInfo: types.RuleInfo{
						SpacePath:  "space",
						RepoPath:   "",
						ID:         2,
						Identifier: "rule2",
						Type:       TypeBranch,
						State:      enum.RuleStateActive,
					},
					Pattern:    []byte(`{"default":true}`),
					Definition: []byte(`{"pullreq":{"approvals":{"require_minimum_count":1}}}`),
				},
			},
			input: MergeVerifyInput{
				Actor:      &types.Principal{ID: 1},
				TargetRepo: &types.Repository{ID: 1, DefaultBranch: "main"},
				PullReq:    &types.PullReq{ID: 1, SourceBranch: "pr", TargetBranch: "main"},
				Method:     enum.MergeMethodRebase,
			},
			expOut: MergeVerifyOutput{
				DeleteSourceBranch: true,
				AllowedMethods:     nil,
			},
			expViol: []types.RuleViolations{
				{
					Rule: types.RuleInfo{
						SpacePath:  "",
						RepoPath:   "space/repo",
						ID:         1,
						Identifier: "rule1",
						Type:       TypeBranch,
						State:      enum.RuleStateActive,
					},
					Bypassed: false,
					Violations: []types.Violation{
						{Code: codePullReqMergeStrategiesAllowed},
					},
				},
				{
					Rule: types.RuleInfo{
						SpacePath:  "space",
						RepoPath:   "",
						ID:         2,
						Identifier: "rule2",
						Type:       TypeBranch,
						State:      enum.RuleStateActive,
					},
					Bypassed: false,
					Violations: []types.Violation{
						{Code: codePullReqApprovalReqMinCount},
					},
				},
			},
		},
		{
			name: "two-rules-merge-strategies",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo: types.RuleInfo{
						SpacePath:  "",
						RepoPath:   "space/repo",
						ID:         1,
						Identifier: "rule1",
						Type:       TypeBranch,
						State:      enum.RuleStateActive,
					},
					Pattern:    []byte(`{"default":true}`),
					Definition: []byte(`{"pullreq":{"merge":{"strategies_allowed":["merge","rebase"]}}}`),
				},
				{
					RuleInfo: types.RuleInfo{
						SpacePath:  "space",
						RepoPath:   "",
						ID:         2,
						Identifier: "rule2",
						Type:       TypeBranch,
						State:      enum.RuleStateActive,
					},
					Pattern:    []byte(`{"default":true}`),
					Definition: []byte(`{"pullreq":{"merge":{"strategies_allowed":["rebase"]}}}`),
				},
			},
			input: MergeVerifyInput{
				Actor:      &types.Principal{ID: 1},
				TargetRepo: &types.Repository{ID: 1, DefaultBranch: "main"},
				PullReq:    &types.PullReq{ID: 1, SourceBranch: "pr", TargetBranch: "main"},
			},
			expOut: MergeVerifyOutput{
				DeleteSourceBranch: false,
				AllowedMethods:     []enum.MergeMethod{enum.MergeMethodRebase},
			},
			expViol: []types.RuleViolations{},
		},
	}

	ctx := context.Background()

	m := NewManager(nil)
	_ = m.Register(TypeBranch, func() Definition {
		return &Branch{}
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := ruleSet{
				rules:   test.rules,
				manager: m,
			}

			out, violations, err := set.MergeVerify(ctx, test.input)
			if err != nil {
				t.Errorf("got error: %s", err.Error())
			}

			if want, got := test.expOut, out; !reflect.DeepEqual(want, got) {
				t.Errorf("output: want=%+v got=%+v", want, got)
			}

			if want, got := len(test.expViol), len(violations); want != got {
				t.Errorf("violations count: want=%d got=%d", want, got)
				return
			}

			for i := range test.expViol {
				if want, got := test.expViol[i].Rule, violations[i].Rule; want != got {
					t.Errorf("violation %d rule: want=%+v got=%+v", i, want, got)
				}
				if want, got := test.expViol[i].Bypassed, violations[i].Bypassed; want != got {
					t.Errorf("violation %d bypassed: want=%t got=%t", i, want, got)
				}
				if want, got := len(test.expViol[i].Violations), len(violations[i].Violations); want != got {
					t.Errorf("violation %d violations count: want=%d got=%d", i, want, got)
					continue
				}
				for j := range test.expViol[i].Violations {
					if want, got := test.expViol[i].Violations[j].Code, violations[i].Violations[j].Code; want != got {
						t.Errorf("violation %d violation %d code: want=%s got=%s", i, j, want, got)
					}
				}
			}
		})
	}
}

func TestIntersectSorted(t *testing.T) {
	tests := []struct {
		name string
		a, b []int
		exp  []int
	}{
		{
			name: "empty",
			a:    []int{},
			b:    []int{},
			exp:  []int{},
		},
		{
			name: "remove last",
			a:    []int{3, 4},
			b:    []int{2, 3},
			exp:  []int{3},
		},
		{
			name: "remove first",
			a:    []int{3, 4, 6},
			b:    []int{4, 5, 6},
			exp:  []int{4, 6},
		},
		{
			name: "remove all",
			a:    []int{3, 4},
			b:    []int{},
			exp:  []int{},
		},
		{
			name: "leave all",
			a:    []int{3, 4},
			b:    []int{1, 2, 3, 4, 5, 6},
			exp:  []int{3, 4},
		},
		{
			name: "remove first and last",
			a:    []int{3, 4, 4, 4, 5},
			b:    []int{4, 6},
			exp:  []int{4, 4, 4},
		},
		{
			name: "remove duplicated",
			a:    []int{3, 4},
			b:    []int{3, 3, 3, 5, 5},
			exp:  []int{3},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if want, got := test.exp, intersectSorted(test.a, test.b); !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v got=%v", want, got)
			}
		})
	}
}
