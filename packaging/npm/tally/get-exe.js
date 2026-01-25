function getExePath() {
  // Detect OS
  // https://nodejs.org/api/process.html#process_process_platform
  let os = process.platform;
  let extension = "";
  if (["win32", "cygwin"].includes(process.platform)) {
    os = "windows";
    extension = ".exe";
  }

  // Detect architecture
  // https://nodejs.org/api/process.html#process_process_arch
  let arch = process.arch;

  return require.resolve(`@contino/tally-${os}-${arch}/bin/tally${extension}`);
}

exports.getExePath = getExePath;
