#!/bin/bash

set -eu

WEB_BUILDER=$1
#export DEV_WEB_BUILDER_OMIT_SLOW_DEPS=1
#export DEV_WEB_BUILDER_UNSAFE_FAST=1

if [ "$WEB_BUILDER" == "vite" ]; then
	pnpm -C client/web exec ts-node -T ../../node_modules/vite/bin/vite.js dev --logLevel silent &
	START=$(gdate +%s%N)
	until curl -s http://localhost:5173/src/enterprise/main.tsx > /dev/null; do 
		sleep 0.05
	done
	END=$(gdate +%s%N)
	DURATION=$(echo "$END - $START" | bc)
	DURATION_MS=$(echo "scale=3; $DURATION/1000000" | bc)
	echo "server: $DURATION_MS ms"
	kill %1
	# wait %1

	TMPDIR=/tmp/vite-tmp-build
	rm -rf "$TMPDIR"
	mkdir "$TMPDIR"
	START=$(gdate +%s%N)
	pnpm -C client/web exec ts-node -T ../../node_modules/vite/bin/vite.js build --logLevel error --outDir "$TMPDIR"
	END=$(gdate +%s%N)
	DURATION=$(echo "$END - $START" | bc)
	DURATION_MS=$(echo "scale=3; $DURATION/1000000" | bc)
	echo "build:  $DURATION_MS ms"
	OUTDIR="$TMPDIR"/assets
	js_assets=$(ls "$OUTDIR"/*.js)
	css_assets=$(ls "$OUTDIR"/*.css)
	echo "js: $(echo $js_assets|wc -w) ($(echo $js_assets|xargs du -hc|tail -n 1|cut -f 1))"
	echo "css: $(echo $css_assets|wc -w) ($(echo $css_assets|xargs du -hc|tail -n 1|cut -f 1))"
elif [ "$WEB_BUILDER" == "esbuild" ]; then
	git clean -fdx client/web/dist > /dev/null 2>&1
	OUTDIR=client/web/dist
	START=$(gdate +%s%N)
	pnpm -C client/web run task:gulp webBuild
	END=$(gdate +%s%N)
	DURATION=$(echo "$END - $START" | bc)
	DURATION_MS=$(echo "scale=3; $DURATION/1000000" | bc)
	echo "build:  $DURATION_MS ms"
	js_assets=$(ls "$OUTDIR"/chunks/*.js "$OUTDIR"/main-*.js)
	css_assets=$(ls "$OUTDIR"/chunks/*.css "$OUTDIR"/main-*.css)
	echo "js: $(echo $js_assets|wc -w) ($(echo $js_assets|xargs du -hc|tail -n 1|cut -f 1))"
	echo "css: $(echo $css_assets|wc -w) ($(echo $css_assets|xargs du -hc|tail -n 1|cut -f 1))"
fi
