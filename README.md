# MagnetarWiki

A Wikidot-compatible wiki engine with a graphical admin console, fine-grained permission groups, and a powerful user level system — built for the F0921 universe.

MagnetarWiki is a fork of [ProjectWikit](https://github.com/WikitTeam/ProjectWikit)
(which itself began as a fork of [RuFoundation](https://github.com/scpru/rufoundation),
the wiki software of the Russian SCP branch). Where ProjectWikit focuses on
faithful Wikidot compatibility and migration, MagnetarWiki goes further:
a wiki engine that any Wikidot community can move to without losing its pages,
history, and the way it works today — while gaining the modern community
management tools Wikidot never offered.

## What it adds on top of ProjectWikit

- **Graphical admin console** — manage the wiki farm, users, and content from a visual backend instead of config files and CLI
- **Fine-grained permission groups** — define custom groups with per-action permissions, beyond the fixed admin/member split
- **User level system** — contribution-driven levels and progression for community members

## What it keeps

- Imports Wikidot sites (users, pages, history, votes, forums) from backups
- Renders Wikidot syntax with output matched against real Wikidot HTML
- Wiki Farm, one instance serves many wikis
- Wikidot account claiming with external verification
- Wikidot syntax parsing via FTML-WIKIT, a hard fork of Wikijump's FTML (see [NOTICE](NOTICE))

## Install

## Documentation

## License

GNU Affero General Public License v3.0 or later. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
As a derivative of ProjectWikit, MagnetarWiki is distributed under the same license.
