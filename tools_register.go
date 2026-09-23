package main

// Blank imports trigger each tool's init(), which registers it with the
// tools registry. Add one line here per new tool package.
import (
	_ "kitaro-rq/internal/tools/jsondiff"
	_ "kitaro-rq/internal/tools/jsonfmt"
	_ "kitaro-rq/internal/tools/jsontocsv"
	_ "kitaro-rq/internal/tools/xmldiff"
	_ "kitaro-rq/internal/tools/xmlfmt"
)
