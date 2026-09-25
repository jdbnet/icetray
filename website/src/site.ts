const githubRepo = import.meta.env.VITE_GITHUB_REPO || 'jdbnet/icetray'
const version = (import.meta.env.VITE_APP_VERSION || '0.0.0').trim()

export const site = {
  name: 'IceTray',
  version,
  githubRepo,
  githubUrl: `https://github.com/${githubRepo}`,
  releasesUrl: `https://github.com/${githubRepo}/releases`,
  releaseBase: `https://github.com/${githubRepo}/releases/download/v${version}`,
  playStoreUrl: 'https://play.google.com/store/apps/details?id=uk.co.jdbnet.icetray',
  aptInstallScript: 'https://apt.jdbnet.co.uk/install/stable.sh',
}

export function artifactUrl(filename: string): string {
  return `${site.releaseBase}/${filename}`
}

export const artifacts = {
  linuxAmd64: 'icetray-linux-amd64',
  linuxArm64: 'icetray-linux-arm64',
  headlessAmd64: 'icetray-headless-linux-amd64',
  headlessArm64: 'icetray-headless-linux-arm64',
  winSetupAmd64: 'icetray-windows-amd64-setup.exe',
  winSetupArm64: 'icetray-windows-arm64-setup.exe',
  winPortableAmd64: 'icetray-windows-amd64.exe',
  winPortableArm64: 'icetray-windows-arm64.exe',
  debAmd64: `icetray_${version}_amd64.deb`,
  debArm64: `icetray_${version}_arm64.deb`,
  androidApk: 'icetray-android.apk',
}
