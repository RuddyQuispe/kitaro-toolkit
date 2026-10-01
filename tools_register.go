package main

// Blank imports trigger each tool's init(), which registers it with the
// tools registry. Add one line here per new tool package.
import (
	_ "kitaro-toolkit/internal/tools/jsondiff"
	_ "kitaro-toolkit/internal/tools/jsonfmt"
	_ "kitaro-toolkit/internal/tools/jsontocsv"
	_ "kitaro-toolkit/internal/tools/xmldiff"
	_ "kitaro-toolkit/internal/tools/xmlfmt"
)
