# Discussion around choosing a Pluggable Architecture

## Goal:

Separate out some business functionality.

## Considerations

- Go Interface We could also add a Go interface so we can fake out the implementation in the open-source version.

## Options

### Leave it at it is

Continue rebasing a code-change on top of the latest open-source main. The code change could be moved out to a dedicated
authenticator in a separate package to make conflicts less likely.

- still maintain internal and external repos
- still maintain internal and external release
- refactoring can help

### Consider Stored procedures

Add a feature to delegate auth to stored procedures in open-source and add some clean-room, not-necessarily production
ready, stored procedures to open-source which pass the acceptance-test suite.

- By choosing this option, we would bring all the SBSS stuff in open source, resulting a complexity and leaking
  unnecessary

### Pluggable Implementation

Add a binary plugin-mechanism (gRPC or otherwise) to open-source that allows for pluggable auth

- pros
    - Separation of code
    - Highly decoupled
    - Language independence
- cons
    - Security aspect: there is more chance that a attacker can come up with customs implementation and execute unwanted
      operations

#### Golang Shared Libraries

- Pros
    - Simple
    - fast because it has no extra processes and IPC serialization
- Cons
    - Package mismatch, change in package code will require to rebuild the whole release e.g.,
    - if Golang compiler version changes, requires rebuilding (re-releasing) the release
    - if transient dependency changes, we require rebuilding(re-releasing)
    - Docs: https://eli.thegreenplace.net/2021/plugins-in-go/

#### Hashicorp Go-Plugin

- Pros
    - Proven, used across hashicorp products/tools e.g., terraform, Nomad, Vault
    - Built-in Logging
    - Cross-language support
    - Plugins are Go interface implementations
    - Stability as it is a separate process
    - Security features on are available e.g., extraction of plugin on filesystem, loading it into memory
    - Docs: https://github.com/hashicorp/go-plugin
- Cons
    - Added complexity to error handling
    - Plugin can stop anytime due to any reason - Health check
    - Multiple processes to manage, adds complexity
    - Involves Inter Process Communication(IPC) reliability
    - Clean up is not straight-forward (requires some steps to perform)

#### Execute it as CLI on command line with known arguments

- Pros
    - Simple
    - Plugin can be built in any language even without RPC over communication
    - No Process management is involved

- Cons
    - Slower: requires process to be started each time e.g, syscalls
    - Added complexity to error handling
    - Simple return parameters parsing are difficult
    - Passing arguments are visible (no encapsulation)
    - Do not consider Golang interface and grap the right one
    - Much larger attack surface for the attacker

### References

- https://cheppers.com/hashicorps-go-plugin-extensive-tutorial
- https://eli.thegreenplace.net/2021/plugins-in-go/
- https://github.com/hashicorp/go-plugin
