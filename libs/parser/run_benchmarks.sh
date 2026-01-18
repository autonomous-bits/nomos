#!/bin/bash
cd /Users/wernerswart/repos/nomos/libs/parser
go test -bench=BenchmarkParse_TypicalConfigWithoutComments -benchmem -run=^$
go test -bench=BenchmarkParse_TypicalConfigWith100Comments -benchmem -run=^$
go test -bench=BenchmarkParse_ConfigWith1000Comments -benchmem -run=^$
