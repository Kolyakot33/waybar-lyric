# Maintainer: Ephemeral <theepehemral.txt@gmail.com>

# shellcheck disable=SC2034
pkgname="ewm-git"
pkgver=r12.41905b5
pkgrel=1
pkgdesc="Ephemeral's Waybar Module"
arch=("x86_64")
url="https://github.com/Nadim147c/EWM"
license=('AGPL')
makedepends=('git' 'go')
optdepends=("wpctl: control volume in pipewire module")
provides=("ewm")
conflicts=("ewm")
source=("$pkgname::git+$url.git")
sha256sums=('SKIP')

pkgver() {
	cd "$pkgname" || return
	printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short=7 HEAD)"
}

build() {
	cd "$pkgname" || return
	go build \
		-ldflags "-w -s -X github.com/Nadim147c/EWM/cmd.Version=$pkgver" \
		-o ./ewmod
}

package() {
	cd "$pkgname" || return
	install -Dm755 ewmod "$pkgdir/usr/bin/ewmod"
	install -Dm644 README.md "$pkgdir/usr/share/doc/$pkgname/README.md"
	install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"

}
