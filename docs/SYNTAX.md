# FlowDoc Syntax

FlowDoc uses indentation to represent nested structures. Indentation is 2 spaces per level. Tabs are replaced with 2 spaces during parsing.

Comments
```
# This is a comment
```

Key-value
```
name = SendWaveHub
retries = 3
enabled = true
```

Strings
```
title = "Hello World"
note = "Supports spaces"
```

Numbers
```
max_users = 1200
pi = 3.14159
```

Booleans
```
debug = true
test_mode = false
```

Objects (Nested)
```
server:
  host = localhost
  port = 8080
```

Arrays
```
regions = [us, eu, asia]
```

Multiline example
```
app:
  name = FlowDoc
  version = 1.0.0

database:
  provider = sqlite
  file = data.db

features:
  enabled = true
  list = [a, b, c]
```
