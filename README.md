# html2vecty
HTML to Vecty compiler! 😎 Transpiles all of your html codes to vecty components in a second! 😎 #move2vecty #html2vecty

> A tool for transpiling html source code to vecty component!

# Installation

Via `go get`:
```shell
go get -v github.com/anikhasibul/html2vecty
```

Via `go install`:
```shell
go install -v github.com/anikhasibul/html2vecty
```

# Usage
From `html2vecty --help`

```txt
html2vecty - A tool for transpiling html source code to vecty component!

Example usage:

1. FROM STDIN: html2vecty can transpile your code from stdin. ex:
	$ cat index.html | html2vecty
or
	$ echo '<h1>Hey Vecty</h1>' | html2vecty

2. FROM FILES: html2vecty can transpile your code from files. ex:
	$ html2vecty index.html

By default html2vecty writes output on stdout. So writing to file is easy!
	$ html2vecty index.html > index.go

Done!

For any kind of query, issue, suggestion,

https://github.com/AnikHasibul/html2vecty
anikhasibul@outlook.com

Thank you!
```

# Stolen?
> Yes!

### From where?
From https://github.com/dave/html2vecty

# Authors

[Anik Hasibul  @AnikHasibul](https://github.com/AnikHasibul) (Maintainer and developer)

[Dave Brophy  @dave](https://github.com/dave) (Original creator)

# Contribution Guidelines

> Welcome! DWTFYW
