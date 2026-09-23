module github.com/jclyons52/eslint-go

go 1.26.5

require (
	github.com/jclyons52/eslint-scope-go v0.0.0
	github.com/jclyons52/espree-go v0.0.0
)

require github.com/jclyons52/acorn-go v0.0.0 // indirect

replace github.com/jclyons52/espree-go => ../espree-go

replace github.com/jclyons52/eslint-scope-go => ../eslint-scope-go

replace github.com/jclyons52/acorn-go => ../acorn-go
