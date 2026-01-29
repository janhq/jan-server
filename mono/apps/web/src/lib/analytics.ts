import posthog from 'posthog-js'

const POSTHOG_KEY = import.meta.env.VITE_POSTHOG_KEY
const POSTHOG_HOST =
  import.meta.env.VITE_POSTHOG_HOST || 'https://app.posthog.com'
const IS_DEV = import.meta.env.DEV
const ENVIRONMENT = IS_DEV ? 'development' : 'production'

export const initAnalytics = () => {
  if (!POSTHOG_KEY) return

  posthog.init(POSTHOG_KEY, {
    api_host: POSTHOG_HOST,
    capture_pageview: true,
    capture_pageleave: true,
    persistence: 'localStorage',
    debug: IS_DEV,
  })
}

export const analytics = {
  identify: (userId: string, properties?: Record<string, unknown>) => {
    posthog.identify(userId, properties)
  },
  reset: () => {
    posthog.reset()
  },

  getUserStatus: (isAuthenticated: boolean): 'guest' | 'authenticated' => {
    return isAuthenticated ? 'authenticated' : 'guest'
  },

  capture: (event: string, properties?: Record<string, unknown>) => {
    posthog.capture(event, {
      platform: 'web',
      environment: ENVIRONMENT,
      ...properties,
    })
  },
}
