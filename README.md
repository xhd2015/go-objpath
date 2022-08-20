# Introduction
Enables easy interaction with go values like struct,map.

Example:
```go
package main

import (
    "github.com/xhd2015/go-objpath"
)

func main(){
    m := map[string]interface{}{
        "A":map[string]interface{}{
            "B":"10",
        },
    }
    objpath.Assert(m,`{"A.B":"10"}`)
    // fail if A.B!=10
}
```

# Assert Syntax
This project introduces a mongodb-like syntax via plain json.

Supported syntax:
```go
{
    // Obj.path.to.inner.prop == 20
    "path.to.innter.prop":20
}

{
    // Obj.path.to.inner.prop > 20
    "path.to.inner.prop":{
        "$gt":"20"
    }
}

{
    // Obj.path.to.*any*.prop > 20
    "path.to.*.prop":{
        "$gt":"20"
    }
}
```

# TODO
add detailed fail reason when one does not match.