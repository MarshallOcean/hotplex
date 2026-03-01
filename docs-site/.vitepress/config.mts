import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'HotPlex',
  description: 'The Strategic Bridge for AI Agent Engineering - Stateful, Secure, and High-Performance.',
  lang: 'en-US',
  base: '/hotplex/',

  head: [
    ['link', { rel: 'icon', href: '/hotplex/favicon.ico' }],
    ['meta', { name: 'theme-color', content: '#00ADD8' }],
    ['meta', { name: 'google', content: 'notranslate' }],
  ],

  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'HotPlex',

    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Ecosystem', link: '/ecosystem/' },
      { text: 'Reference', link: '/reference/architecture' },
      { text: 'Blog', link: '/blog/' },
      { text: 'GitHub', link: 'https://github.com/hrygo/hotplex' }
    ],


    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          collapsed: false,
          items: [
            { text: 'Getting Started', link: '/guide/getting-started' },
            { text: 'What is HotPlex?', link: '/guide/introduction' },
          ]
        },
        {
          text: 'Core Concepts',
          collapsed: false,
          items: [
            { text: 'Architecture Overview', link: '/guide/architecture' },
            { text: 'State Management', link: '/guide/state' },
            { text: 'Hooks System', link: '/guide/hooks' },
          ]
        },
        {
          text: 'User Guides',
          collapsed: false,
          items: [
            { text: 'ChatApps Overview', link: '/guide/chatapps' },
            { text: 'Slack Integration', link: '/guide/chatapps-slack' },
            { text: 'Observability', link: '/guide/observability' },
            { text: 'Production Deployment', link: '/guide/deployment' },
          ]
        }
      ],
      '/reference/': [
        {
          text: 'Development Reference',
          items: [
            { text: 'System Architecture', link: '/reference/architecture' },
            { text: 'API Reference', link: '/reference/api' },
            { text: 'Protocol Specification', link: '/reference/protocol' },
            { text: 'Hooks API', link: '/reference/hooks-api' },
          ]
        },
        {
          text: 'SDKs',
          items: [
            { text: 'Go SDK', link: '/sdks/go-sdk' },
            { text: 'Python SDK', link: '/sdks/python-sdk' },
            { text: 'TypeScript SDK', link: '/sdks/typescript-sdk' },
          ]
        }
      ],
      '/blog/': [
        {
          text: 'Updates & Engineering',
          items: [
            { text: 'Latest Updates', link: '/blog/' },
            { text: 'Roadmap 2026', link: '/blog/roadmap-2026' },
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/hrygo/hotplex' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 HotPlex Team'
    },

    search: {
      provider: 'local'
    },

    editLink: {
      pattern: 'https://github.com/hrygo/hotplex/edit/main/docs-site/:path',
      text: 'Edit this page on GitHub'
    },

    lastUpdated: {
      text: 'Last updated',
      formatOptions: {
        dateStyle: 'medium',
        timeStyle: 'short'
      }
    }
  }
})

