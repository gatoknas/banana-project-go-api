# GitHub Projects, Branching & Issue Tracking Rule (Go API)

## Mandate
All non-trivial feature implementations, bug fixes, architectural spikes, and epic roadmaps must be tied to GitHub Project #6 ([Projects Board #6](https://github.com/users/gatoknas/projects/6)), which serves as the **Single Unified Board** for the Banana Project ecosystem (`banana-project-go-api`, `banana-project-web`, `banana-project-mobile`, `banana-project-bruno-collection`).

## Rules & Quality Gates

### Gate 1: Issue-First Approval Protocol (MANDATORY)
1. **Formulation**: Groom requirements, acceptance criteria, and technical scope into user stories.
2. **Issue Creation on GitHub**: Create GitHub issues in the respective repository, apply the repository label (`repo:api`), and link them to Project #6 with status `Ready` / `Todo`.
3. **URL Presentation & Approval Gate**: Display the complete list of clickable GitHub Issue URLs in the chat. **STOP and wait for explicit user approval before writing any code.**

### Gate 2: Branch-First Creation & Checkout (MANDATORY)
1. **Trigger**: Occurs immediately after receiving user approval and before modifying any files.
2. **Selective Repository Rule**: Only create a branch if this repository requires code modifications.
3. **Pull Latest from Main**:
   ```bash
   git checkout main
   git pull origin main
   ```
4. **Branch Naming Standard**: `<issue-number>-<slugified-issue-purpose>` (e.g. `9-revenue-summary-and-statistics-endpoint`).
5. **Checkout**:
   ```bash
   git checkout -b <issue-number>-<slugified-issue-purpose>
   ```
6. **Card Status**: Move the corresponding card on Project #6 to `In Progress`.
7. **Verification**: Verify `git branch --show-current` confirms you are on the issue branch before modifying source files.

### Gate 3: Pull Request Creation & Review (MANDATORY)
1. **Trigger**: Occurs once code implementation and tests are complete and committed to the issue branch.
2. **Push to Remote**:
   ```bash
   git push -u origin <issue-number>-<slugified-issue-purpose>
   ```
3. **Mandatory PR Template Content**:
   - **Title**: `feat(api): #<issue> - <purpose>`
   - **Linked Issue**: `Closes #<issue>`
   - **Scope of Changes**: Context, architectural decisions, and key capabilities.
   - **List of Files Affected**: Table of new and modified files.
   - **Evidence of Unit Tests Passing**: Full terminal test logs showing 0 failures.
   - **Coverage Report**: Exact package/statement coverage percentages (`go test -cover ...`).
   - **Security Audit**: Confirmation that `python scripts/secret_scanner.py` passed with 0 detected secrets.
4. **Board Update**: Transition card on Project #6 from `In Progress` to **`In Review`**.
5. **Presentation**: Display clickable PR URLs in chat and pause for user code review and merge.
6. **Completion**: Once merged into `main`, GitHub closes the issue and the card transitions to **`Done`**.

---

## Repository Label Mandate
Every card on Project #6 must have its corresponding repository label:
- `repo:api` for Go backend tasks
- `repo:web` for Vue.js frontend tasks
- `repo:mobile` for Android mobile tasks
- `repo:bruno` for Bruno API collection tasks

---

## Secret Prevention
- Under no circumstances should sensitive tokens, credentials, or keys be pasted into issue bodies, PR descriptions, or committed to git. Follow the project's [secret-prevention rule](file:///c:/Users/danie/Documents/SourceCode/banana-project/banana-project-go-api/.agents/rules/secret-prevention.md).
