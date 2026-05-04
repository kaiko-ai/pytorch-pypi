# Sonatype PyTorch PyPI Improved Mirror

<!-- Badges Section -->
[![shield_gh-workflow-test]][link_gh-workflow-test]
[![shield_license]][license_file]
<!-- Add other badges or shields as appropriate -->

---

This project provides a [PyTorch](https://pytorch.org/get-started/locally/) Python index mirror that adds necessary strict PEP 503 format requirements permitting to use it with [Nexus Repository](https://help.sonatype.com/en/pypi-repositories.html#download--search--and-install-packages-using-pip) — or directly with `pip`, `uv`, and `poetry`.

The index is regenerated daily via GitHub Actions from <https://download.pytorch.org/whl/> and published to GitHub Pages at <https://sonatype-nexus-community.github.io/pytorch-pypi/whl/>. Wheels themselves are not re-hosted; the generated `index.html` files link back to PyTorch's CDN.

- [Sonatype PyTorch PyPI Improved Mirror](#sonatype-pytorch-pypi-improved-mirror)
  - [Available indexes](#available-indexes)
  - [Direct usage (pip / uv / poetry)](#direct-usage-pip--uv--poetry)
  - [Usage with Nexus Repository](#usage-with-nexus-repository)
  - [Development](#development)
  - [The Fine Print](#the-fine-print)

## Available indexes

Base URL: `https://sonatype-nexus-community.github.io/pytorch-pypi/whl/`

| Variant         | Index URL suffix          | Notes                              |
|-----------------|---------------------------|------------------------------------|
| All packages    | `simple/`                 | Full mirror, all platforms         |
| CPU only        | `cpu/simple/`             |                                    |
| CPU (cxx11 ABI) | `cpu-cxx11-abi/simple/`   |                                    |
| CUDA 11.8       | `cu118/simple/`           |                                    |
| CUDA 12.1       | `cu121/simple/`           |                                    |
| CUDA 12.4       | `cu124/simple/`           |                                    |
| CUDA 12.6       | `cu126/simple/`           |                                    |
| CUDA 12.8       | `cu128/simple/`           |                                    |
| CUDA 12.9       | `cu129/simple/`           |                                    |
| CUDA 13.0       | `cu130/simple/`           |                                    |
| ROCm 6.0–6.4    | `rocm6.X/simple/`         | `rocm6.0`, `rocm6.1`, …, `rocm6.4` |
| Intel XPU       | `xpu/simple/`             |                                    |
| Nightly         | `nightly/simple/`         | Pre-release builds                 |

The authoritative list of currently published variants is browsable at <https://sonatype-nexus-community.github.io/pytorch-pypi/whl/>.

## Direct usage (pip / uv / poetry)

### `pip`

```sh
pip install torch --index-url https://sonatype-nexus-community.github.io/pytorch-pypi/whl/cu129/simple
```

### `uv`

```sh
uv pip install torch --index-url https://sonatype-nexus-community.github.io/pytorch-pypi/whl/cu129/simple
```

Or in `pyproject.toml`:

```toml
[[tool.uv.index]]
name = "pytorch-cu129"
url = "https://sonatype-nexus-community.github.io/pytorch-pypi/whl/cu129/simple"
```

### Poetry

```toml
[[tool.poetry.source]]
name = "pytorch-cu129"
url = "https://sonatype-nexus-community.github.io/pytorch-pypi/whl/cu129/simple"
priority = "explicit"
```

## Usage with Nexus Repository

As a [Nexus Repository](https://help.sonatype.com/en/sonatype-nexus-repository.html) admin, configure a [new PyPI proxy](https://help.sonatype.com/en/pypi-repositories.html#proxying-pypi-repositories) repository:
- define name: `pypi-pytorch` for example
- define URL for remote storage: `https://sonatype-nexus-community.github.io/pytorch-pypi/whl/`

You can also create compute platform specific proxies if necessary, like `pypi-pytorch-cu129` pointing to `https://sonatype-nexus-community.github.io/pytorch-pypi/whl/cu129/`

For nightly versions, simply create a `pypi-pytorch-nightly` repository pointing to `https://sonatype-nexus-community.github.io/pytorch-pypi/whl/nightly/`

These PyPI proxies can also be added to your `pypi-all` group repository if you created one.

### Access With `pip`

Newly created PyPI proxies can now be used with `pip` using provided `--index-url`, like:

```
pip3 install torch==2.8.0+cu129 --index-url http://localhost:8081/repository/pypi-pytorch/simple
```
or
```
pip3 install torch --index-url http://localhost:8081/repository/pypi-pytorch-cu129/simple
```
for nightly
```
pip3 install torch --index-url http://localhost:8081/repository/pypi-pytorch-nightly/simple

or

pip3 install torch==2.8.0.dev20250613+cu129 http://localhost:8081/repository/pypi-pytorch-nightly/simple
```

## Development

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## The Fine Print

Remember:

This project is part of the [Sonatype Nexus Community](https://github.com/sonatype-nexus-community) organization, which is not officially supported by Sonatype. Please review the latest pull requests, issues, and commits to understand this project's readiness for contribution and use.

* File suggestions and requests on this repo through GitHub Issues, so that the community can pitch in
* Use or contribute to this project according to your organization's policies and your own risk tolerance
* Don't file Sonatype support tickets related to this project— it won't reach the right people that way

Last but not least of all - have fun!

<!-- Links Section -->
[shield_gh-workflow-test]: https://img.shields.io/github/actions/workflow/status/sonatype-nexus-community/pytorch-pypi/update.yml?branch=main&logo=GitHub&logoColor=white "build"
[shield_license]: https://img.shields.io/github/license/sonatype-nexus-community/pytorch-pypi?logo=open%20source%20initiative&logoColor=white "license"

[link_gh-workflow-test]: https://github.com/sonatype-nexus-community/pytorch-pypi/actions/workflows/update.yml?query=branch%3Amain
[license_file]: https://github.com/sonatype-nexus-community/pytorch-pypi/blob/main/LICENSE
