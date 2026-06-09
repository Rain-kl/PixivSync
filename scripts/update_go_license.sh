#!/usr/bin/env bash

set -euo pipefail

MODE="write"

usage() {
  cat <<'EOF'
Usage:
  scripts/update_go_license.sh [--check]

Updates Go source files with the project license header.

Rules:
  - Existing Apache license headers gain:
      Modified by Arctel.net, 2026
    directly after the Copyright line.
  - Go files without a license header receive an Arctel.net Apache header.
  - //go:build and legacy // +build constraints stay at the top of the file.

Options:
  --check   Report files that would change and exit non-zero if any are found.
EOF
}

while (($#)); do
  case "$1" in
    --check)
      MODE="check"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

changed=0

find_go_files() {
  find . \
    \( -path './.*' \
    -o -path './docs' \
    -o -path './frontend/node_modules' \
    -o -path './frontend/.next' \
    -o -path './frontend/out' \
    -o -path './internal/router/dist' \
    -o -path './vendor' \) -prune \
    -o -type f -name '*.go' -print
}

process_file() {
  local file="$1"
  local out="$2"

  perl -0 - "$file" "$out" <<'PERL'
use strict;
use warnings;

my ($file, $out) = @ARGV;

open my $in, '<', $file or die "open $file: $!";
local $/;
my $src = <$in>;
close $in;

my $modified = 'Modified by Arctel.net, 2026';
my $new_header = <<'HEADER';
/*
Copyright 2026 Arctel.net

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
HEADER
chomp $new_header;

my @lines = split /\n/, $src, -1;
my @prefix;
my $i = 0;

if (@lines && ($lines[0] =~ m{^//go:build } || $lines[0] =~ m{^// \+build })) {
    while ($i < @lines) {
        if ($lines[$i] =~ m{^//go:build } || $lines[$i] =~ m{^// \+build }) {
            push @prefix, $lines[$i++];
            next;
        }
        if ($lines[$i] eq '') {
            push @prefix, $lines[$i++];
            last if $i >= @lines || ($lines[$i] !~ m{^//go:build } && $lines[$i] !~ m{^// \+build });
            next;
        }
        last;
    }
}

my $prefix = @prefix ? join("\n", @prefix) . "\n" : '';
my $body = join "\n", @lines[$i .. $#lines];
my $result;

if ($body =~ m{\A(/\*.*?\*/)(\n*)}s) {
    my $block = $1;
    my $rest = substr($body, length($1) + length($2));

    if ($block =~ /Licensed under the Apache License, Version 2\.0/ && $block =~ /Copyright /) {
        if ($block !~ /Copyright [^\n]*Arctel\.net/ && $block !~ /\Q$modified\E/) {
            $block =~ s/^(Copyright [^\n]*)$/$1\n$modified/m;
        }
        $result = $prefix . $block . "\n\n" . $rest;
    } else {
        $rest =~ s/\A\n+//;
        $result = $prefix . $new_header . "\n\n" . $block . "\n\n" . $rest;
    }
} else {
    $body =~ s/\A\n+//;
    $result = $prefix . $new_header . "\n\n" . $body;
}

open my $fh, '>', $out or die "write $out: $!";
print {$fh} $result;
close $fh;
PERL
}

while IFS= read -r file; do
  tmp_file="$tmp_dir/${file#./}"
  mkdir -p "$(dirname "$tmp_file")"
  process_file "$file" "$tmp_file"

  if ! cmp -s "$file" "$tmp_file"; then
    changed=1
    if [[ "$MODE" == "check" ]]; then
      echo "needs license update: $file"
    else
      cp "$tmp_file" "$file"
      echo "updated: $file"
    fi
  fi
done < <(find_go_files)

if [[ "$MODE" == "check" && "$changed" -ne 0 ]]; then
  exit 1
fi
