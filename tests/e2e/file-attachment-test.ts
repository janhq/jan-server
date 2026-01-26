import { chromium, type Page, type Browser } from "playwright";
import * as fs from "fs";
import * as path from "path";

const BASE_URL = process.env.BASE_URL || "http://localhost:3001";
const EMAIL = process.env.TEST_EMAIL || "nguyen@jan.ai";
const PASSWORD = process.env.TEST_PASSWORD || "nguyen";

async function login(page: Page) {
  console.log("Navigating to login page...");
  await page.goto(BASE_URL);

  // Wait for either login form or already logged in state
  await page.waitForTimeout(2000);

  // Check if we need to log in
  const emailInput = page.locator('input[name="email"], input[type="email"]');
  if (await emailInput.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log("Logging in...");
    await emailInput.fill(EMAIL);

    const passwordInput = page.locator(
      'input[name="password"], input[type="password"]'
    );
    await passwordInput.fill(PASSWORD);

    const submitButton = page.locator(
      'button[type="submit"], button:has-text("Sign in"), button:has-text("Log in")'
    );
    await submitButton.click();

    // Wait for redirect after login
    await page.waitForTimeout(3000);
  }

  console.log("Logged in successfully");
}

async function createTestFile(filename: string, content: string): Promise<string> {
  const testDir = path.join(__dirname, "test-files");
  if (!fs.existsSync(testDir)) {
    fs.mkdirSync(testDir, { recursive: true });
  }
  const filePath = path.join(testDir, filename);
  fs.writeFileSync(filePath, content);
  return filePath;
}

async function testFileAttachment(page: Page) {
  console.log("\n=== Testing File Attachment ===");

  // Create a simple test file (text file for testing)
  const testFilePath = await createTestFile(
    "test-document.txt",
    "This is a test document content for OCR testing.\n\nIt contains multiple lines of text.\n\nThe quick brown fox jumps over the lazy dog."
  );

  // Find the file input or attach button
  console.log("Looking for file attachment button...");

  // Look for attachment button/trigger
  const attachButton = page.locator(
    'button:has-text("Add photos"), button:has-text("Add files"), button[aria-label*="attach"], [data-testid="attach-button"]'
  );

  if (await attachButton.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log("Found attach button, clicking...");
    await attachButton.click();
    await page.waitForTimeout(1000);
  }

  // Find file input
  const fileInput = page.locator('input[type="file"]');
  if (await fileInput.count() > 0) {
    console.log("Uploading test file...");
    await fileInput.setInputFiles(testFilePath);
    await page.waitForTimeout(2000);
  } else {
    console.log("No file input found, trying drag and drop...");
  }

  // Type a message
  const textInput = page.locator(
    'textarea[placeholder*="message"], textarea[placeholder*="Message"], [contenteditable="true"]'
  );
  if (await textInput.isVisible({ timeout: 5000 }).catch(() => false)) {
    await textInput.fill("Please summarize this document");
  }

  // Submit
  const submitButton = page.locator(
    'button[type="submit"]:not([disabled]), button:has-text("Send"):not([disabled]), button[aria-label="Send"]'
  );
  if (await submitButton.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log("Submitting message...");
    await submitButton.click();
  }

  // Wait for response
  console.log("Waiting for response...");
  await page.waitForTimeout(10000);

  // Check for errors
  const errorMessage = page.locator(
    '.error, [role="alert"], :text("error"), :text("not supported")'
  );
  if (await errorMessage.isVisible({ timeout: 5000 }).catch(() => false)) {
    const errorText = await errorMessage.textContent();
    console.error("ERROR FOUND:", errorText);
    return false;
  }

  console.log("Test completed successfully");
  return true;
}

async function main() {
  console.log("Starting file attachment test...");
  console.log(`Base URL: ${BASE_URL}`);

  const browser: Browser = await chromium.launch({
    headless: false, // Set to true for CI
    slowMo: 100,
  });

  const context = await browser.newContext({
    viewport: { width: 1280, height: 720 },
  });

  const page = await context.newPage();

  // Listen for console messages to capture errors
  page.on("console", (msg) => {
    if (msg.type() === "error") {
      console.log(`[Browser Error]: ${msg.text()}`);
    }
  });

  // Listen for request failures
  page.on("requestfailed", (request) => {
    console.log(`[Request Failed]: ${request.url()} - ${request.failure()?.errorText}`);
  });

  // Listen for responses with errors
  page.on("response", async (response) => {
    if (response.status() >= 400) {
      try {
        const body = await response.text();
        console.log(`[HTTP ${response.status()}]: ${response.url()}`);
        console.log(`Response body: ${body.substring(0, 500)}`);
      } catch (e) {
        // Ignore
      }
    }
  });

  try {
    await login(page);
    await testFileAttachment(page);
  } catch (error) {
    console.error("Test failed with error:", error);
  } finally {
    // Keep browser open for debugging
    console.log("\nTest finished. Browser will stay open for inspection.");
    console.log("Press Ctrl+C to close.");
    await new Promise(() => {}); // Keep running
  }
}

main().catch(console.error);
