---
title: Search
author: Fazal Majid
type: page
date: 2008-08-19T03:18:23+00:00
url: /search
searchpage: true
---

{{ if .Results }}
{{ range $_, $result := .Results }}
<h2><a href="{{ $result.Path }}">{{ $result.Title }}</a></h2>
<p>{{ $result.Summary }}</p>
{{ end }}
{{ else }}
<p>No search results found.</p>
{{ end }}
