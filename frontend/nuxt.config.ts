export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui'
  ],

  devtools: {
    enabled: true
  },

  app: {
    pageTransition: { name: 'page', mode: 'out-in' }
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    public: {
      apiBase: 'http://127.0.0.1:8080/api/v1'
    }
  },

  routeRules: {
    '/': { prerender: true }
  },

  compatibilityDate: '2025-01-15',

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
