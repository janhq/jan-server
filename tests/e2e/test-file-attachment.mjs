#!/usr/bin/env node

import { chromium } from "playwright";
import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BASE_URL = process.env.BASE_URL || "http://localhost:3001";
const EMAIL = process.env.TEST_EMAIL || "nguyen@jan.ai";
const PASSWORD = process.env.TEST_PASSWORD || "nguyen";

async function login(page) {
  console.log("Navigating to login page...");
  await page.goto(BASE_URL);

  // Wait for page load
  await page.waitForTimeout(3000);

  // Check if we need to log in
  const emailInput = page.locator('input[name="username"], input[type="email"], input[name="email"]');
  if (await emailInput.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log("Logging in with credentials...");
    await emailInput.fill(EMAIL);

    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await passwordInput.fill(PASSWORD);

    // Click submit button
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();

    // Wait for redirect after login
    await page.waitForTimeout(5000);
    console.log("Logged in successfully");
  } else {
    console.log("Already logged in or no login form found");
  }
}

async function createTestPptxFile() {
  // Create a minimal PPTX file (it's just a ZIP with specific structure)
  // For testing, we'll create a simple text file with PPTX extension
  // The real test is whether the system properly handles it
  const testDir = path.join(__dirname, "test-files");
  if (!fs.existsSync(testDir)) {
    fs.mkdirSync(testDir, { recursive: true });
  }

  // Create a simple text file for testing document upload
  const txtFilePath = path.join(testDir, "test-document.txt");
  fs.writeFileSync(txtFilePath, `Test Document Content

This is a test document for OCR testing.
It contains multiple lines of text.

The quick brown fox jumps over the lazy dog.
Testing document attachment feature.
`);

  return txtFilePath;
}

async function testFileAttachment(page) {
  console.log("\n=== Testing File Attachment ===");

  // Create test file
  const testFilePath = await createTestPptxFile();
  console.log(`Created test file: ${testFilePath}`);

  // Look for attach button or file input
  console.log("Looking for file input...");

  // The file input might be hidden, find it
  const fileInput = page.locator('input[type="file"]').first();

  if (await fileInput.count() > 0) {
    console.log("Found file input, uploading test file...");
    await fileInput.setInputFiles(testFilePath);
    await page.waitForTimeout(3000);
    console.log("File uploaded");
  } else {
    console.log("No file input found directly. Looking for attach button...");

    // Try to find the attach button
    const attachButton = page.getByText("Add photos or files").first();
    if (await attachButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await attachButton.click();
      await page.waitForTimeout(1000);
    }
  }

  // Type a message
  console.log("Typing message...");
  const textInput = page.locator('textarea').first();
  if (await textInput.isVisible({ timeout: 5000 }).catch(() => false)) {
    await textInput.fill("Please summarize this document");
    await page.waitForTimeout(1000);
  }

  // Take screenshot before submit
  await page.screenshot({ path: path.join(__dirname, "before-submit.png") });
  console.log("Screenshot saved: before-submit.png");

  // Find and click submit button
  console.log("Submitting message...");
  const submitButton = page.locator('button[type="submit"]').first();
  if (await submitButton.isEnabled({ timeout: 5000 }).catch(() => false)) {
    await submitButton.click();
    console.log("Message submitted");
  } else {
    // Try pressing Enter
    await textInput.press("Enter");
  }

  // Wait for response
  console.log("Waiting for response...");
  await page.waitForTimeout(15000);

  // Take screenshot after
  await page.screenshot({ path: path.join(__dirname, "after-submit.png") });
  console.log("Screenshot saved: after-submit.png");

  // Check for error messages
  const pageContent = await page.content();
  const hasError = pageContent.includes("not supported") ||
                   pageContent.includes("media type") ||
                   pageContent.includes("Error");

  if (hasError) {
    console.error("ERROR: Found error message in page content!");
    // Check for specific error
    const errorRegex = /file part media type.*not supported/i;
    if (errorRegex.test(pageContent)) {
      console.error("SPECIFIC ERROR: File part media type not supported");
    }
    return false;
  }

  console.log("Test completed - no obvious errors detected");
  return true;
}

async function main() {
  console.log("Starting file attachment test...");
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Email: ${EMAIL}`);

  const browser = await chromium.launch({
    headless: process.env.HEADLESS !== "false",
    slowMo: 50,
  });

  const context = await browser.newContext({
    viewport: { width: 1280, height: 720 },
  });

  const page = await context.newPage();

  // Listen for console messages
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      console.log(`[Browser Error]: ${msg.text()}`);
    }
  });

  // Listen for responses with errors
  page.on("response", async (response) => {
    if (response.status() >= 400) {
      try {
        const body = await response.text().catch(() => "");
        console.log(`[HTTP ${response.status()}]: ${response.url()}`);
        if (body) {
          console.log(`Response: ${body.substring(0, 500)}`);
        }
      } catch (e) {
        // Ignore
      }
    }
  });

  let success = false;
  try {
    await login(page);
    success = await testFileAttachment(page);
  } catch (error) {
    console.error("Test failed with error:", error);
  } finally {
    if (process.env.KEEP_OPEN !== "true") {
      await browser.close();
    } else {
      console.log("\nBrowser kept open for inspection. Press Ctrl+C to close.");
      await new Promise(() => {});
    }
  }

  process.exit(success ? 0 : 1);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
