// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package session_test

import (
	. "github.com/pingcap/check"
	gofail "github.com/pingcap/gofail/runtime"
	"github.com/powerispower/tidb/util/testkit"
)

func (s *testSessionSuite) TestFailStatementCommit(c *C) {
	defer gofail.Disable("github.com/powerispower/tidb/session/mockStmtCommitError")

	tk := testkit.NewTestKitWithInit(c, s.store)
	tk.MustExec("create table t (id int)")
	tk.MustExec("begin")
	tk.MustExec("insert into t values (1)")
	gofail.Enable("github.com/powerispower/tidb/session/mockStmtCommitError", `return(true)`)
	_, err := tk.Exec("insert into t values (2)")
	c.Assert(err, NotNil)
	tk.MustExec("commit")
	tk.MustQuery(`select * from t`).Check(testkit.Rows("1"))
}
