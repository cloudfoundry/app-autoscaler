## Code Quality
utoscaler uses Pre-commit for keeping our code base clean. Upon a git commit, it checks .go and .java files for formatting.
We recommend to use our guidelines. These style only applies to .go and java files.

- Java Formatter : Google Styles are used (https://github.com/google/google-java-format)
- Go Formatter:  Standard Go formatter (go fmt) is used

### Install Pre-Commit

Install https://pre-commit.com/ on your developer machine/laptop
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
git add <files>
git commit -m <message>

```bash
$ git commit -m "add basic auth" 
[WARNING] Unstaged files detected.
[INFO] Stashing unstaged files to /Users/I545443/.cache/pre-commit/patch1626333779-50304.
java-formatter...........................................................Failed
- hook id: java-formatter
- exit code: 2

/Users/I545443/sap/asalan316/app-autoscaler/.cache
/Users/I545443/sap/asalan316/app-autoscaler/.cache
download path : /Users/I545443/sap/asalan316/app-autoscaler/.cache/google-java-format-1.10.0-all-deps.jar
Formatter Results:
Analyzing java file(s) using google-java-format-1.10.0-all-deps.jar...
Incorrect formatting found:
scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/rest/ControllerExceptionHandler.java
Please correct the formatting of the files(s) using one of the following options:
  - Reformat Code - IntelliJ or Format Document - Eclipse (google code style required)
  - java -jar /Users/I545443/sap/asalan316/app-autoscaler/.cache/google-java-format-1.10.0-all-deps.jar -replace scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/rest/ControllerExceptionHandler.java

go imports...............................................................Failed
- hook id: go-imports
- exit code: 1
- files were modified by this hook

src/autoscaler/eventgenerator/aggregator/aggregator.go

go fmt...................................................................Passed
[WARNING] Stashed changes conflicted with hook auto-fixes... Rolling back fixes...
[INFO] Restored changes from /Users/I545443/.cache/pre-commit/patch1626333779-50304.

```
- Example Output of correctly formatted code
```
$ git commit -m "add basic auth"                                                                            
[WARNING] Unstaged files detected.
[INFO] Stashing unstaged files to /Users/I545443/.cache/pre-commit/patch1626333979-52031.
java-formatter.......................................(no files to check)Skipped
go imports...........................................(no files to check)Skipped
go fmt...............................................(no files to check)Skipped
[INFO] Restored changes from /Users/I545443/.cache/pre-commit/patch1626333979-52031.
[code-style-guide 3c1c7d1] add basic auth
 3 files changed, 40 insertions(+), 1 deletion(-)
 create mode 100644 style-guide/README.md
 rename style-guide/{inspect-java-format.sh => inspect-java-format-0.1.sh} (100%)

```

## Importing Java Style Guide in IDE 
Having styles configures in IDE help developers to focus on business logic (instead of formatting issue). For this purpose, enabling formatting increases productivity.
Since, App-Autoscaler's scheduler component is written in java, it makes sense to enable java formatter only for scheduler project.

- Open scheduler as project in the IDE. Doing so, will only apply Google style in scheduler projec)
- Download the style from this link https://raw.githubusercontent.com/google/styleguide/gh-pages/intellij-java-google-style.xml
- Intellij,
  - Under Preferences -> Editor -> Code Style -> Java. There is Scheme settings (settings icon on right side) -> import schemes-> intellij idea code style xml and select current scheme. Current Scheme will only import code style for scheduler project.
  - the reformat code can be use to auto format files using already configured code style. 
- Eclipse,
    - open Eclipse -> Preferences(or Settings). In the search, type “formatter”, and select the Java -> Code Style -> Formatter menu item -> Import

