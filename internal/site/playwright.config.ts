import { defineConfig } from '@playwright/test'

const baseURL = process.env.MONITOR_TEST_URL || 'http://127.0.0.1:8091'
if (!['127.0.0.1', 'localhost'].includes(new URL(baseURL).hostname)) throw new Error('E2E tests may only run against an isolated local monitor instance.')
export default defineConfig({
 testDir: './tests', testMatch:'monitor.spec.ts', timeout: 30000, workers: 1, fullyParallel: false,
 use: {baseURL, viewport:{width:1440,height:1000}, headless:true, launchOptions:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE ? {executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE} : {}},
 reporter: [['list']],
})
