// Copyright 2017 PingCAP, Inc.
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

package executor

import (
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/tidb/domain"
	"github.com/pingcap/tidb/statistics"
	"github.com/pingcap/tidb/store/tikv/oracle"
	"github.com/pingcap/tidb/types"
)

func (e *ShowExec) fetchShowStatsMeta() error {
	do := domain.GetDomain(e.ctx)
	h := do.StatsHandle()
	dbs := do.InfoSchema().AllSchemas()
	for _, db := range dbs {
		for _, tbl := range db.Tables {
			statsTbl := h.GetTableStats(tbl)
			if !statsTbl.Pseudo {
				e.appendRow([]interface{}{
					db.Name.O,
					tbl.Name.O,
					e.versionToTime(statsTbl.Version),
					statsTbl.ModifyCount,
					statsTbl.Count,
				})
			}
		}
	}
	return nil
}

func (e *ShowExec) fetchShowStatsHistogram() error {
	do := domain.GetDomain(e.ctx)
	h := do.StatsHandle()
	dbs := do.InfoSchema().AllSchemas()
	for _, db := range dbs {
		for _, tbl := range db.Tables {
			statsTbl := h.GetTableStats(tbl)
			if !statsTbl.Pseudo {
				for _, col := range statsTbl.Columns {
					e.histogramToRow(db.Name.O, tbl.Name.O, col.Info.Name.O, 0, col.Histogram, col.AvgColSize(statsTbl.Count))
				}
				for _, idx := range statsTbl.Indices {
					e.histogramToRow(db.Name.O, tbl.Name.O, idx.Info.Name.O, 1, idx.Histogram, 0)
				}
			}
		}
	}
	return nil
}

func (e *ShowExec) histogramToRow(dbName string, tblName string, colName string, isIndex int, hist statistics.Histogram, avgColSize float64) {
	e.appendRow([]interface{}{
		dbName,
		tblName,
		colName,
		isIndex,
		e.versionToTime(hist.LastUpdateVersion),
		hist.NDV,
		hist.NullCount,
		avgColSize,
	})
}

func (e *ShowExec) versionToTime(version uint64) types.Time {
	t := time.Unix(0, oracle.ExtractPhysical(version)*int64(time.Millisecond))
	return types.Time{Time: types.FromGoTime(t), Type: mysql.TypeDatetime}
}

func (e *ShowExec) fetchShowStatsBuckets() error {
	do := domain.GetDomain(e.ctx)
	h := do.StatsHandle()
	dbs := do.InfoSchema().AllSchemas()
	for _, db := range dbs {
		for _, tbl := range db.Tables {
			statsTbl := h.GetTableStats(tbl)
			if !statsTbl.Pseudo {
				for _, col := range statsTbl.Columns {
					err := e.bucketsToRows(db.Name.O, tbl.Name.O, col.Info.Name.O, 0, col.Histogram)
					if err != nil {
						return errors.Trace(err)
					}
				}
				for _, idx := range statsTbl.Indices {
					err := e.bucketsToRows(db.Name.O, tbl.Name.O, idx.Info.Name.O, len(idx.Info.Columns), idx.Histogram)
					if err != nil {
						return errors.Trace(err)
					}
				}
			}
		}
	}
	return nil
}

// bucketsToRows converts histogram buckets to rows. If the histogram is built from index, then numOfCols equals to number
// of index columns, else numOfCols is 0.
func (e *ShowExec) bucketsToRows(dbName, tblName, colName string, numOfCols int, hist statistics.Histogram) error {
	isIndex := 0
	if numOfCols > 0 {
		isIndex = 1
	}
	for i := 0; i < hist.Len(); i++ {
		lowerBoundStr, err := statistics.ValueToString(hist.GetLower(i), numOfCols)
		if err != nil {
			return errors.Trace(err)
		}
		upperBoundStr, err := statistics.ValueToString(hist.GetUpper(i), numOfCols)
		if err != nil {
			return errors.Trace(err)
		}
		e.appendRow([]interface{}{
			dbName,
			tblName,
			colName,
			isIndex,
			i,
			hist.Buckets[i].Count,
			hist.Buckets[i].Repeat,
			lowerBoundStr,
			upperBoundStr,
		})
	}
	return nil
}

func (e *ShowExec) fetchShowStatsHealthy() {
	do := domain.GetDomain(e.ctx)
	h := do.StatsHandle()
	dbs := do.InfoSchema().AllSchemas()
	for _, db := range dbs {
		for _, tbl := range db.Tables {
			statsTbl := h.GetTableStats(tbl)
			if !statsTbl.Pseudo {
				var healthy int64
				if statsTbl.ModifyCount < statsTbl.Count {
					healthy = int64((1.0 - float64(statsTbl.ModifyCount)/float64(statsTbl.Count)) * 100.0)
				} else if statsTbl.ModifyCount == 0 {
					healthy = 100
				}
				e.appendRow([]interface{}{
					db.Name.O,
					tbl.Name.O,
					healthy,
				})
			}
		}
	}
}
