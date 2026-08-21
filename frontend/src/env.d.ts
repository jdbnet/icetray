/// <reference types="vite/client" />

interface Window {
  go: {
    main: {
      App: Record<string, (...args: any[]) => Promise<any>>
    }
  }
  runtime: {
    EventsOn: (eventName: string, callback: (...data: any[]) => void) => () => void
    EventsOff: (eventName: string, ...additionalEventNames: string[]) => void
  }
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
