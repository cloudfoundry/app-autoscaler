# Spike Suggestions

## Considerations

- Go Interface We could also add a Go interface so we can fake out the implementation in the open-source version.

## Options

### Leave it at it is

Continue rebasing a code-change on top of the latest open-source main. The code change could be moved out to a dedicated
authenticator in a separate package to make conflicts less likely.

- still maintain internal and external repos
- still maintain internal and external release
- refactoring can help

## Consider Stored procedures

Add a feature to delegate auth to stored procedures in open-source and add some clean-room, not-necessarily production
ready, stored procedures to open-source which pass the acceptance-test suite.

- By choosing this option, we would bring all the SBSS stuff in open source, resulting a complexity and leaking
  unnecessary

### Pluggable Implementation

Add a binary plugin-mechanism (gRPC or otherwise) to open-source that allows for pluggable auth

#### Golang Shared Libraries

- Pros
    - Simple,
    - fast because it has no extra processes and IPC serialization
 
- Cons
    - Package mismatch, change in package code will require to rebuild the whole release e.g.,
    - if Golang compiler version changes, requires rebuilding (re-releasing) the release
    - if transient dependency changes, we require rebuilding(re-releasing)

https://eli.thegreenplace.net/2021/plugins-in-go/

#### Hashicorp Go-Plugin

- Pros
- Cons

#### Execute it as CLI on command line with known arguments

- Pros
- Cons


