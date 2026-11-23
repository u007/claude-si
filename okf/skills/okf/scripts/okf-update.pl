#!/usr/bin/perl
# okf-update.pl: index and log maintenance for an OKF v0.1 bundle.
#
#   perl okf-update.pl <docs-dir> index
#   perl okf-update.pl <docs-dir> log <Creation|Update|Deprecation|Deletion> <doc-path> <summary>
#
# Both modes run under an exclusive flock(2) on <docs-dir>/.okf.lock (the same
# lock ocode's knowledge.WithBundleLock takes). `log` appends one entry to
# log.md, then regenerates index.md. `index` only regenerates index.md.
# The index has the same lines as ocode's knowledge.GenerateIndex; only entries
# whose titles tie (e.g. title-less docs) may order differently, since Go's sort is unstable.
# Perl core only: stock macOS and Linux have it, python3 and flock(1) may be absent.
use strict;
use warnings;
use Fcntl qw(:flock);
use File::Find;
use File::Temp qw(tempfile);
use POSIX qw(strftime);

my ($root, $mode, @rest) = @ARGV;
die "usage: okf-update.pl <docs-dir> index | log <Action> <doc-path> <summary>\n"
  unless defined $mode && ($mode eq 'index' || ($mode eq 'log' && @rest == 3));
$root =~ s{/+$}{};
my $index = "$root/index.md";
open my $ifh, '<', $index or die "okf: cannot read $index: $!\n";
my $index_raw = do { local $/; <$ifh> };
close $ifh;
my $version = okf_version($index_raw);
die "okf: $index has no okf_version marker; not a bundle\n" if $version eq '';

open my $lock, '>>', "$root/.okf.lock" or die "okf: cannot open $root/.okf.lock: $!\n";
local $SIG{ALRM} = sub { die "okf: timed out after 60s waiting for $root/.okf.lock\n" };
alarm 60;
flock $lock, LOCK_EX or die "okf: flock $root/.okf.lock: $!\n";
alarm 0;

if ($mode eq 'log') {
  my ($action, $path, $summary) = @rest;
  die "okf: action must be Creation, Update, Deprecation or Deletion\n"
    unless $action =~ /^(Creation|Update|Deprecation|Deletion)$/;
  $path =~ s{^/+}{};
  die "okf: doc path must stay inside the bundle\n" if $path =~ m{(^|/)\.\.(/|$)};
  die "okf: doc path must not be index.md or log.md\n" if $path eq 'index.md' || $path eq 'log.md';
  die "okf: $root/$path does not exist\n" if $action ne 'Deletion' && !-f "$root/$path";
  die "okf: summary must be one line\n" if $summary =~ /\n/;
  append_log($action, $path, $summary);
}
write_index();
print "okf: index.md regenerated\n";
exit 0;

# --- index ---------------------------------------------------------------

sub okf_version {
  my ($s) = @_;
  return '' unless $s =~ /^---\r?\n/;
  for my $line (split /\n/, substr($s, index($s, "\n") + 1)) {
    (my $t = $line) =~ s/^\s+|\s+$//g;
    last if $t eq '---' || $t eq '...';
    if ($t =~ /^okf_version:(.*)$/) {
      (my $v = $1) =~ s/^\s+|\s+$//g;
      $v =~ s/^["' ]+|["' ]+$//g;
      return $v;
    }
  }
  return '';
}

# Go filepath.Match semantics: whole-string match, * and ? never cross '/'.
sub glob_re {
  my ($p) = @_;
  my $re = '';
  for (my $i = 0; $i < length $p; $i++) {
    my $c = substr($p, $i, 1);
    if    ($c eq '*') { $re .= '[^/]*' }
    elsif ($c eq '?') { $re .= '[^/]' }
    elsif ($c eq '[') {
      my $j = index($p, ']', $i + 1);
      if ($j < 0) { $re .= '\['; next }
      my $cls = substr($p, $i + 1, $j - $i - 1);
      $cls =~ s/^\^/!/;
      $cls =~ s/^!/^/;
      $re .= "[$cls]";
      $i = $j;
    }
    elsif ($c eq '\\' && $i + 1 < length $p) { $re .= quotemeta(substr($p, ++$i, 1)) }
    else { $re .= quotemeta($c) }
  }
  return qr/^$re$/;
}

sub write_index {
  my @defaults = map { glob_re($_) } ('PLAN-*.md', '*.OCODE.md');
  my (@ignore_file, @ignore_dir);
  if (open my $fh, '<', "$root/.okfignore") {
    while (my $l = <$fh>) {
      $l =~ s/^\s+|\s+$//g;
      next if $l eq '' || $l =~ /^#/;
      push @ignore_file, glob_re($l);
      (my $d = $l) =~ s{/$}{};
      push @ignore_dir, glob_re($d);
    }
    close $fh;
  }

  my @paths;
  find({
    no_chdir => 1,
    preprocess => sub { sort @_ },
    wanted => sub {
      my $full = $File::Find::name;
      return if $full eq $root;
      (my $rel = $full) =~ s{^\Q$root\E/}{};
      (my $name = $rel) =~ s{.*/}{};
      if (-d $full) {
        if ($name =~ /^\./ || grep { $name =~ $_ || $rel =~ $_ } @ignore_dir) {
          $File::Find::prune = 1;
        }
        return;
      }
      return if $name =~ /^\./ || $name !~ /\.md$/;
      return if $name eq 'index.md' || $name eq 'log.md';
      return if grep { $name =~ $_ } @defaults;
      return if grep { $name =~ $_ || $rel =~ $_ } @ignore_file;
      push @paths, $rel;
    },
  }, $root);

  my (%sections, @unclassified);
  for my $rel (sort @paths) {
    open my $fh, '<', "$root/$rel" or die "okf: cannot read $root/$rel: $!\n";
    my $raw = do { local $/; <$fh> };
    close $fh;
    my $fm = frontmatter($raw);
    if (!$fm) { push @unclassified, $rel; next }
    my $dir = $rel =~ m{^([^/]+)/} ? $1 : '';
    push @{ $sections{$dir} }, {
      path => $rel,
      title => $fm->{title} // '',
      desc => $fm->{description} // '',
      deprecated => (($fm->{status} // '') eq 'deprecated'),
    };
  }

  my $out = "---\nokf_version: $version\n---\n\n";
  for my $dir (sort keys %sections) {
    $out .= '# ' . ($dir eq '' ? 'Concepts' : $dir) . "\n\n";
    for my $e (sort { lc($a->{title}) cmp lc($b->{title}) || $a->{path} cmp $b->{path} } @{ $sections{$dir} }) {
      my $title = $e->{title} eq '' ? $e->{path} : $e->{title};
      $out .= "- [$title]($e->{path})";
      $out .= " - $e->{desc}" if $e->{desc} ne '';
      $out .= ' (deprecated)' if $e->{deprecated};
      $out .= "\n";
    }
    $out .= "\n";
  }
  if (@unclassified) {
    $out .= "# Unclassified\n\n";
    for my $rel (@unclassified) {
      (my $base = $rel) =~ s{.*/}{};
      $out .= "- [$base]($rel)\n";
    }
    $out .= "\n";
  }

  my ($tfh, $tmp) = tempfile('index.md.tmp.XXXXXX', DIR => $root);
  binmode $tfh;
  print {$tfh} $out or die "okf: write $tmp: $!\n";
  close $tfh or die "okf: close $tmp: $!\n";
  rename $tmp, $index or do { unlink $tmp; die "okf: rename $tmp to $index: $!\n" };
}

# Returns a hashref of top-level scalar keys, or undef when the file is not
# conforming (no closed frontmatter block, or the block is not a key mapping).
sub frontmatter {
  my ($raw) = @_;
  return undef unless $raw =~ /^---\r?\n/;
  my $rest = substr($raw, index($raw, "\n") + 1);
  my $block;
  if ($rest =~ /^(.*?)\n---\r?(?:\n|$)/s) { $block = $1 }
  elsif ($rest =~ /^---/) { $block = '' }
  else { return undef }
  $block =~ s/\r//g;

  my (%fm, $key, @buf);
  my $flush = sub { $fm{$key} = scalar_value(@buf) if defined $key; @buf = () };
  for my $line (split /\n/, $block, -1) {
    if ($line =~ /^([^\s#][^:]*):(?:\s+(.*?))?\s*$/ || $line =~ /^([^\s#][^:]*):$/) {
      $flush->();
      $key = $1;
      @buf = (defined $2 ? $2 : '');
    } elsif ($line =~ /^\s*(#.*)?$/ && !defined $key) {
      next;
    } elsif (defined $key) {
      push @buf, $line;
    } else {
      return undef;
    }
  }
  $flush->();
  return %fm ? \%fm : undef;
}

# Minimal YAML scalar decoding for the styles bundle docs use: plain, '…',
# "…", and | or > block scalars. Sequences and maps decode to '' (unused).
sub scalar_value {
  my ($first, @more) = @_;
  if ($first =~ /^([|>])([+-]?)\d*\s*$/) {
    my ($style, $chomp) = ($1, $2);
    my @lines = @more;
    my ($ind) = map { /^(\s*)/; length $1 } grep { /\S/ } @lines;
    $ind //= 0;
    @lines = map { length($_) >= $ind ? substr($_, $ind) : '' } @lines;
    pop @lines while @lines && $lines[-1] eq '';
    my $v;
    if ($style eq '|') { $v = join "\n", @lines }
    else {
      $v = '';
      for my $i (0 .. $#lines) {
        my $l = $lines[$i];
        if ($i == 0) { $v = $l; next }
        if ($l eq '') { $v .= "\n" }
        elsif ($lines[$i - 1] eq '' || $l =~ /^\s/ || $lines[$i - 1] =~ /^\s/) { $v .= ($v =~ /\n$/ ? '' : "\n") . $l }
        else { $v .= " $l" }
      }
    }
    return $chomp eq '-' ? $v : "$v\n";
  }
  my $text = fold($first, @more);
  return '' if $text =~ /^[\[{]/ || $text eq '' && @more && $more[0] =~ /^\s*-/;
  if ($text =~ /^'(.*)'\s*(#.*)?$/s) { (my $v = $1) =~ s/''/'/g; return $v }
  if ($text =~ /^"(.*)"\s*(#.*)?$/s) {
    my $v = $1;
    my %esc = (n => "\n", t => "\t", r => "\r", '"' => '"', '\\' => '\\', '/' => '/', 0 => "\0", ' ' => ' ');
    $v =~ s/\\(.)/exists $esc{$1} ? $esc{$1} : "\\$1"/ge;
    return $v;
  }
  $text =~ s/\s+#.*$//;
  return $text;
}

# YAML line folding for flow scalars: one line break becomes a space, each
# extra empty line becomes a newline.
sub fold {
  my @lines = map { (my $l = $_) =~ s/^\s+|\s+$//g; $l } @_;
  pop @lines while @lines > 1 && $lines[-1] eq '';
  my $v = shift @lines;
  my $blank = 0;
  for my $l (@lines) {
    if ($l eq '') { $blank++; next }
    $v .= $blank ? "\n" x $blank : ' ';
    $v .= $l;
    $blank = 0;
  }
  return $v;
}

# --- log -----------------------------------------------------------------

sub append_log {
  my ($action, $path, $summary) = @_;
  my $log = "$root/log.md";
  my $today = strftime('%Y-%m-%d', localtime);
  my $entry = "* **$action**: $summary ([$path](/$path))";
  my @lines;
  if (-e $log) {
    open my $fh, '<', $log or die "okf: cannot read $log: $!\n";
    @lines = map { chomp; $_ } <$fh>;
    close $fh;
  } else {
    @lines = ('# Directory Update Log');
  }
  my ($today_at) = grep { $lines[$_] =~ /^##\s+\Q$today\E\s*$/ } 0 .. $#lines;
  if (defined $today_at) {
    my $at = $today_at + 1;
    $at++ while $at < @lines && $lines[$at] =~ /^\s*$/;
    splice @lines, $at, 0, $entry;
  } else {
    my ($title) = grep { $lines[$_] =~ /^# Directory Update Log/ } 0 .. $#lines;
    die "okf: $log has no '# Directory Update Log' heading\n" unless defined $title;
    my $at = $title + 1;
    $at++ while $at < @lines && $lines[$at] =~ /^\s*$/;
    my @block = ('', "## $today", '', $entry);
    push @block, '' if $at < @lines;
    splice @lines, $title + 1, $at - $title - 1, @block;
  }
  my ($tfh, $tmp) = tempfile('log.md.tmp.XXXXXX', DIR => $root);
  print {$tfh} join("\n", @lines), "\n" or die "okf: write $tmp: $!\n";
  close $tfh or die "okf: close $tmp: $!\n";
  rename $tmp, $log or do { unlink $tmp; die "okf: rename $tmp to $log: $!\n" };
}
