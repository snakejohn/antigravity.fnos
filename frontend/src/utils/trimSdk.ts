import { TrimApp } from '@trimjs/web-app'

let sdkInstance: TrimApp | null = null

function getSdk(): TrimApp {
  if (!sdkInstance) {
    sdkInstance = new TrimApp()
  }
  return sdkInstance
}

export const trimSdk = {
  get sdk(): TrimApp {
    return getSdk()
  },
  get isStandaloneWeb(): boolean {
    try {
      return Boolean(this.sdk.isStandaloneWeb)
    } catch {
      return true
    }
  },
  get isWeb(): boolean {
    try {
      return Boolean(this.sdk.isWeb)
    } catch {
      return false
    }
  },
  async ready(): Promise<void> {
    try {
      await this.sdk.ready()
    } catch {
      // standalone browser
    }
  },
  async openFileManager(path: string): Promise<{ success: boolean; message?: string }> {
    await this.ready()
    if (this.isStandaloneWeb) {
      return { success: false, message: `路径已复制，请在飞牛系统桌面内使用文件管理器打开: ${path}` }
    }
    try {
      await this.sdk.openFileManager(path)
      return { success: true }
    } catch (e: unknown) {
      return { success: false, message: (e as Error)?.message || String(e) }
    }
  },
  async openURL(url: string, target = '_blank'): Promise<void> {
    await this.ready()
    try {
      await this.sdk.openURL(url, target)
    } catch {
      window.open(url, target)
    }
  },
  async setTitle(title: string): Promise<void> {
    document.title = title
    try {
      await this.sdk.setTitle(title)
    } catch {
      // ignore
    }
  },
  async initPlatformTheme(onThemeChange: (theme: 'dark' | 'light') => void): Promise<void> {
    await this.ready()
    try {
      const config = await this.sdk.getPlatformConfig()
      if (config?.theme) {
        onThemeChange(config.theme)
      }
      if (this.isWeb && !this.isStandaloneWeb) {
        await this.sdk.$on('os/theme', (theme: 'dark' | 'light') => onThemeChange(theme))
      }
    } catch {
      // ignore outside fnOS desktop
    }
  }
}
