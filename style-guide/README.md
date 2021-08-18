## Code Quality
Autoscaler uses [Pre-commit](https://pre-commit.com/) for keeping our code base clean. It is a package manager fo git hooks.
Upon a git commit, it checks .go and .java files for potential code problems.

Two static code analyzers are used as git-hooks which are defined in the .pre-commit-config file.

- Java Formatter : Checkstyle and Google formatter are used along-with [Google Styles](https://github.com/google/google-java-format)
- Golangci:  [Golangci-lint](https://github.com/golangci/golangci-lint) is used

We recommend to install pre-commit on develop machines. This will help to catch issue before code review.

### Install Pre-Commit

Install [Pre-commit](https://pre-commit.com/) on developer laptop
- using pip
```bash
pip install pre-commit
```
- using curl
```bash
curl https://pre-commit.com/install-local.py | python -
```

- using homebrew
```bash
brew install pre-commit
```
- using conda 
```bash
conda install -c conda-forge pre-commit
```

## Usage
```
git add <files>
git commit -m <message>
```

## Real World Example

### Commit changes from Local

```bash
$ git commit  -m "aas82 fix GHA linter:  Golang and Java - local"
[WARNING] Unstaged files detected.
[INFO] Stashing unstaged files to /Users/<USER>/.cache/pre-commit/patch1629118042-59848.
java-formatter...........................................................Failed
- hook id: java-formatter
- exit code: 1

Running Checkstyle using <APP_AUTOSCALER_REPO>/.cache/$CHECKSTYLE_JAR_NAME...
[WARN] <APP_AUTOSCALER_REPO>/scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/util/DataValidationHelper.java:19:19: Abbreviation in name 'LINTTT_CHECK' must contain no more than '1' consecutive capital letters. [AbbreviationAsWordInName]
[WARN] <APP_AUTOSCALER_REPO>scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/util/DataValidationHelper.java:19:19: Member name 'LINTTT_CHECK' must match pattern '^[a-z][a-z0-9][a-zA-Z0-9]*$'. [MemberName]

golangci-lint............................................................Failed
- hook id: golangci-lint
- exit code: 2

<APP_AUTOSCALER_REPO>/src/autoscaler
golangci-lint run
api/brokerserver/broker_handler.go:24: File is not `gofmt`-ed with `-s` (gofmt)
              logger          lager.Logger
make: *** [lint] Error 1

[INFO] Restored changes from /Users/<USER>/.cache/pre-commit/patch1629118042-59848.

```
In the above output, Some issues has been reported:

**Go File:** incorrect formatting in api/brokerserver/broker_handler.go:24 (reported by Golangci-lint)

**Java:** Incorrect variable name reported by Java-formatter

To fix them: 

For Go: 
```
gofmt -s -w api/brokerserver/broker_handler.go

```
For Java, just correct the variable name

Upon committing again, Golangci-lint passed but java-formatter has reported some formatting problems:
```bash
$ git commit  -m "aas82 fix GHA linter:  Golang and Java - local"                                                                              
[WARNING] Unstaged files detected.
[INFO] Stashing unstaged files to /Users/<USER>/.cache/pre-commit/patch1629118377-61629.
java-formatter...........................................................Failed
- hook id: java-formatter
- exit code: 2

Running Checkstyle using <APP_AUTOSCALER_REPO>/.cache/CHECKSTYLE_JAR_NAME...
============================================================
Google Formatting using <APP_AUTOSCALER_REPO>/.cache/google-java-format-1.11.0-all-deps.jar...
Incorrect formatting found:
Please correct the formatting of the files(s) using one of the following options:
   java --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED  -jar <APP_AUTOSCALER_REPO>/.cache/google-java-format-1.11.0-all-deps.jar -replace --skip-javadoc-formatting scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/util/DataValidationHelper.java

golangci-lint............................................................Passed
[INFO] Restored changes from /Users/<USER>/.cache/pre-commit/patch1629118377-61629.

```

To fix them, just execute the command as suggested by the java-formatter. It will auto-format the java sources.
```bash
java --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED \
   --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED  
  -jar <APP_AUTOSCALER_REPO>/.cache/google-java-format-1.11.0-all-deps.jar \
  -replace --skip-javadoc-formatting scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/util/DataValidationHelper.java

```

## Successful Commit

```bash
$ git commit  -m "aas82 fix GHA linter:  Golang and Java - local"                                      
[WARNING] Unstaged files detected.
[INFO] Stashing unstaged files to /Users/<USER>/.cache/pre-commit/patch1629118875-64154.
java-formatter...........................................................Passed
golangci-lint............................................................Passed
[INFO] Restored changes from /Users/<USER>/.cache/pre-commit/patch1629118875-64154.
[aas82-verify-linters cff357fa] aas82 fix GHA linter:  Golang and Java - local
 3 files changed, 8 insertions(+), 4 deletions(-)

```

### Skip Pre-Commit Git Hook Locally
`git commit --no-verify -m "<COMMI_MESSAGE"`


**Note:** The same static code analyzers are used via GitHub Actions. 