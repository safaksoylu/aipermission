#!/usr/bin/env bash

set -Eeuo pipefail

# --------------------------------------------------
# AYARLAR
# --------------------------------------------------

FORK_REPO="git@github.com:safaksoylu/aipermission.git"
UPSTREAM_REPO="https://github.com:aipermission/aipermission.git"

BRANCH="main"
SYNC_BRANCH="automatic-upstream-sync"
WORK_DIR="${HOME}/Projects/safaksoylu-git/aipermission"

LOCK_FILE="/tmp/fork-sync-FORK_REPO.lock"

log() {
    printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

abort_merge() {
    if git rev-parse --verify -q MERGE_HEAD >/dev/null 2>&1; then
        git merge --abort || true
    fi
}

on_error() {
    local exit_code=$?

    if [[ -d "${WORK_DIR}/.git" ]]; then
        cd "$WORK_DIR"
        abort_merge
    fi

    log "Senkronizasyon başarısız oldu."
    exit "$exit_code"
}

trap on_error ERR

# Aynı anda birden fazla çalışmasını engelle.
exec 9>"$LOCK_FILE"

if ! flock -n 9; then
    log "Başka bir senkronizasyon işlemi zaten çalışıyor."
    exit 0
fi

mkdir -p "$(dirname "$WORK_DIR")"

# Repository daha önce klonlanmadıysa fork'u klonla.
if [[ ! -d "${WORK_DIR}/.git" ]]; then
    log "Fork repository klonlanıyor..."
    git clone "$FORK_REPO" "$WORK_DIR"
fi

cd "$WORK_DIR"

git remote set-url origin "$FORK_REPO"

if git remote get-url upstream >/dev/null 2>&1; then
    git remote set-url upstream "$UPSTREAM_REPO"
else
    git remote add upstream "$UPSTREAM_REPO"
fi

# Commit edilmemiş yerel değişiklik varsa hiçbir şeyi silme.
if [[ -n "$(git status --porcelain)" ]]; then
    log "Çalışma dizininde commit edilmemiş değişiklikler var."
    log "Güvenlik nedeniyle senkronizasyon durduruldu."
    exit 1
fi

log "Fork ve upstream bilgileri alınıyor..."
git fetch --prune origin
git fetch --prune upstream

# Her zaman fork'taki güncel main dalını taban al.
#
# Bu işlem origin/main'i değiştirmez ve senin commitlerini silmez.
# Yalnızca geçici senkronizasyon dalını origin/main'e getirir.
git switch --force-create "$SYNC_BRANCH" "origin/${BRANCH}"

BEFORE_COMMIT="$(git rev-parse HEAD)"

log "upstream/${BRANCH}, fork'taki commitlerin üzerine merge ediliyor..."

if ! git merge --no-edit "upstream/${BRANCH}"; then
    log "Merge conflict oluştu."
    log "Senin commitlerin korunmuştur; origin/${BRANCH} değiştirilmedi."
    abort_merge
    exit 1
fi

AFTER_COMMIT="$(git rev-parse HEAD)"

if [[ "$BEFORE_COMMIT" == "$AFTER_COMMIT" ]]; then
    log "Yeni upstream commit'i bulunamadı. Fork zaten güncel."
    exit 0
fi

log "Merge sonucu fork'un ${BRANCH} dalına gönderiliyor..."

# Bilerek --force kullanılmıyor.
# Bu sırada başka biri main'e commit gönderdiyse push güvenli biçimde başarısız olur.
git push origin "HEAD:${BRANCH}"

log "Senkronizasyon başarıyla tamamlandı."
log "Önceki fork commit'i: ${BEFORE_COMMIT}"
log "Yeni merge commit'i:  ${AFTER_COMMIT}"