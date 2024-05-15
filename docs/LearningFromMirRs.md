# Learnings from Mirrs

- Change message bus scope
  - Previously:
    - Message bus for device only and apis for system interaction
  - Now:
    - Message bus for all interactions. A unique endpoint and
    more similar apps pattern. Full Event system driven

- Good amount of documentation and diagrams of the systems. Combined a lot of it into one Document. Help be more organize. Also gave ways to a lot of thinking and a v2 of the message bus architecture of events, topics, publishers and subscribers.

- Better toolings and egornomics
  - better toolings to support the project such as:
    - CLI
    - TUI
    - Various scripts
  - better ergonomics to help enjoy the project:
    - tmux layout with all required infra started
    - integration and unit testing
    - docker compose for required infra

- Trying to be more vertical in the sens of working on all pieces at the same time for a x ffeature instead of doing a lot of features on one piece and then too much to catch up else where.

- Less abstractions in code.  Trying too hard instead of
just repeating some code. Wonder if having a boilerplate generator
for how I do my apps. Could be in bash

-  Taking more my time to apply myself. Can be hard to fully polish especially early on. So much to do and eager to get fonctionnality in. Breaking the ice in all the systems is hard. Coding can be like recipes, so the first few apps are the recipes on how to do the different interactions with the system, testing, ui, etc. Lot of things and effort to get all of that setup. But it's worth taking the time because they will be the funding blocks that can be copy pasted over and over.

- Rust async just doesn't work. The fuck is this. Feel like Python trying to be efficient.
