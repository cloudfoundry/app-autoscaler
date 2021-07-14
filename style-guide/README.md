## Code Quality
utoscaler uses Pre-commit for keeping our code base clean. Upon a git commit, it checks .go and .java files for formatting.
We recommend to use our guidelines. These style only applies to .go and java files.

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

- Output
```
[INFO] Stashing unstaged files to /Users/I545443/.cache/pre-commit/patch1626333399-49087.
java-formatter.......................................(no files to check)Skipped
go imports...........................................(no files to check)Skipped
go fmt...............................................(no files to check)Skipped
[INFO] Restored changes from /Users/I545443/.cache/pre-commit/patch1626333399-49087.

```

