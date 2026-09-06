// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  // Static site generation for GitHub Pages
  ssr: false,

  app: {
    baseURL: '/wee/',
    head: {
      title: 'Wee',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'The coolest cats code in sandboxes.' },

        // OpenGraph
        { property: 'og:site_name', content: 'wee.cat' },
        { property: 'og:title', content: 'The coolest cats code in sandboxes | wee.cat' },
        { property: 'og:description', content: 'The coolest cats code in sandboxes.' },
        { property: 'og:type', content: 'website' },
        { property: 'og:url', content: 'https://wee.cat/' },
        { property: 'og:image', content: 'https://wee.cat/og-image.png' },
        { property: 'og:image:width', content: '1200' },
        { property: 'og:image:height', content: '630' },
        { property: 'og:image:alt', content: 'wee.cat - The coolest cats code in sandboxes' },

        // Twitter Card
        { name: 'twitter:card', content: 'summary_large_image' },
        { name: 'twitter:title', content: 'The coolest cats code in sandboxes | wee.cat' },
        { name: 'twitter:description', content: 'The coolest cats code in sandboxes.' },
        { name: 'twitter:image', content: 'https://wee.cat/og-image.png' },
        { name: 'twitter:image:alt', content: 'wee.cat - The coolest cats code in sandboxes' }
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/wee/favicon.ico' },
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap' }
      ]
    }
  },
})
