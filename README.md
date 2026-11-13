# <img src="data/icons/vinegar.svg" alt="Vinegar logo" height="64px"> Vinegar

[![Pipeline Status][pipeline_img    ]][pipeline    ]
[![Version        ][version_img     ]][version     ]
[![Flathub        ][flathub_img     ]][flathub     ]
[![Report Card    ][goreportcard_img]][goreportcard]
[![Discord Server ][discord_img     ]][discord     ]

An open-source, configurable, fast bootstrapper for running Roblox Studio on Linux.

[pipeline]:     https://github.com/vinegarhq/vinegar/actions/workflows/build.yml
[pipeline_img]: https://img.shields.io/github/actions/workflow/status/vinegarhq/vinegar/build.yml?style=flat-square&label=build%20%26%20tests
[version]:     https://github.com/vinegarhq/vinegar/releases/latest
[version_img]: https://img.shields.io/github/v/release/vinegarhq/vinegar?style=flat-square&display_name=tag
[flathub]:     https://flathub.org/apps/details/org.vinegarhq.Vinegar
[flathub_img]: https://img.shields.io/flathub/downloads/org.vinegarhq.Vinegar?style=flat-square
[goreportcard]:     https://goreportcard.com/report/github.com/vinegarhq/vinegar
[goreportcard_img]: https://goreportcard.com/badge/github.com/vinegarhq/vinegar?style=flat-square
[discord]:     https://discord.gg/dzdzZ6Pps2
[discord_img]: https://img.shields.io/discord/1069506340973707304?style=flat-square&label=discord

![Showcase of Vinegar's manager and bootstrapper window](data/screenshots/vinegar.png)

# About this fork
This is an unofficial fork of [vinegarhq/vinegar](https://github.com/vinegarhq/vinegar)
with fixes for problems found while running Roblox Studio on a hybrid-GPU laptop.
It is not affiliated with VinegarHQ, and it is licensed under GPL-3.0 like the original.

| Change | What it does |
|---|---|
| Keyboard fix | Stops Wine's `winebus` from opening `/dev/hidraw*`, which made some USB keyboards reset every ~1.25 s while Studio ran. Opt back in with `studio.hidraw = true`. A brand-new prefix is covered from its second launch. |
| Log and installer pruning | Keeps the newest Studio logs (at most 20 files / 256 MiB) and removes stale WebView installers. Studio logs had reached 3 GB. |
| Stale DXVK cleanup | Removes DXVK DLLs left next to Studio's executable when a different renderer is selected. |
| Pre-launch checks | Warns about `WINEDLLOVERRIDES` that disable parts of Studio (e.g. `mimalloc-redirect`) and about low free VRAM. |
| GPU selection | Prefers the discrete GPU by `boot_vga` rather than by card order, which changes between boots on hybrid laptops. |

Known limits: Wine/DXVK/Vulkan crashes, Studio's own CPU and RAM use, and
Studio's Auto Recovery prompt are outside what these changes can fix.

There are no prebuilt packages. Build from source with Go and `make`; see the
upstream documentation for dependencies. The fixes are split into one branch
each (`fix/disable-wine-hidraw`, `feat/prune-studio-data`,
`feat/preflight-checks`, `fix/stale-dxvk`) so they can be taken separately.

# See Also
+ [Discord server][discord]
+ [Documentation](https://vinegarhq.github.io)
+ [Roblox-Studio-Mod-Manager](https://github.com/MaximumADHD/Roblox-Studio-Mod-Manager)
+ [Bloxstrap](https://github.com/pizzaboxer/bloxstrap)

# Acknowledgements
+ Credits to
  + [pizzaboxer](https://github.com/pizzaboxer)
  + [MaximumADHD](https://github.com/MaximumADHD)
