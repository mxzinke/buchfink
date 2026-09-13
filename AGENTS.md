# Buchfink

Buchfink is a software for making book-keeping simple enough for small german businesses to do it theirself.

The language of the software is german, the code is still mostly in English whereby we use German documentation and code comments.

## Documenting

We do use code comments and other markdown files to explain background information, decesions we took or legal references/requirements. Every comment can go stale or contradict the code. Keep minimal and document logics, requirements or background information; default to deleting a comment if it tells about things which have been done in the past. Only ever document the current state.

If a lot of documentation is needed, it's a sign for bad code, naming or wrong archtitecture; rewrite or restructure in clear modules and interfaces instead.

## Commits

Use Conventional Commits for both commit messages and PR titles: `type(scope): summary` — e.g. `fix(webui): …`, `feat(runtime): …`, `refactor(controller): …`. The **commit body** is where the _why_ belongs — decisions, rationale and trade-offs go in commit body rather than in code comments.