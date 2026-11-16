# CHANGE_HISTORY

Commit history organized by date (newest first, based on git log).

## 2026-11-12

- fix(app): reuse explorer window when opening a file's folder

## 2026-11-11

- Update README, update go and libraries version.

## 2026-11-10

- Update go and wails version

## 2026-11-09

- feat(scanner): filter by project .gitignore and dedup dir selection by hierarchy
- feat(redaction): move sensitive filter config to project settings with rule profiles
- feat(editor): add in-app directory tree multi-select for include directories
- feat(editor): move extension scan to include patterns, add file picker and clear-all

## 2026-11-08

- feat(editor): flush-left labels, output extension dropdown and tag list quick-fill buttons

## 2026-11-06

- feat: select exported file in file manager and reset stale workbench result

## 2026-11-05

- Update docs.
- feat(compress): add dropMarkedComments option and overhaul comment stripping — strip comments by language syntax, keep license/TODO-marked blocks
- Fix an UI issue.
- Update screenshot.

## 2026-11-04

- fix(ui): align project card buttons, readable build stamp, language sync
- feat(ui): replace language select with a dropdown menu in the title bar
- feat(ui): rework visual design to a restrained tool look

## 2026-11-01

- feat(ui): refine workbench/editor UI, reuse explorer window, add build stamp
- feat(settings): add global settings dialog with export encoding
- chore: add .gitattributes to normalize line endings — normalize line endings: LF by default for text files, CRLF for Windows scripts, binary assets untouched
- feat: add token-saving smart export and app icon tooling — token-saving smart export mode and app icon generation tooling

## 2026-10-24

- Update screenshot and README

## 2026-10-14

- feat(ui): add open-config-dir button and help menu

## 2026-10-08

- Support localization.
- feat(export): add whitespace-minified output and dry-run preview; fix scan/editor issues
- Fix preview issue, update screenshot.
- support output split into files. — split output into multiple files by token limit
- Fix path not exist issue. Enhance new project wizard.

## 2026-10-07

- Project init.
- feat: two-view GUI redesign and build pipeline cleanup
- feat: WYSIWYG project editor dialog with live YAML preview
- Update README.md
