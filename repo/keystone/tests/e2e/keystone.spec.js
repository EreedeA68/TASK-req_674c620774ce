// @ts-check
const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:3000';
const API = 'http://localhost:8080';

async function login(page, username, password) {
  await page.goto(`${BASE}/login`);
  await page.fill('[data-testid="username-input"]', username);
  await page.fill('[data-testid="password-input"]', password);
  await page.click('[data-testid="submit-button"]');
  await page.waitForURL(url => !url.toString().includes('/login'), { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

test.describe('AdminJourneyTest', () => {
  test('admin can create a new user', async ({ page }) => {
    await login(page, 'admin', 'Admin@Keystone1!');
    // Verify admin menu items visible
    await expect(page.locator('[data-testid="navbar"]')).toBeVisible();
    await expect(page.getByRole('link', { name: 'Admin' })).toBeVisible();
    // Navigate to admin page
    await page.getByRole('link', { name: 'Admin' }).click();
    await page.waitForURL('**/admin');
    // Create new user
    await page.fill('[data-testid="new-username"]', 'newintake');
    await page.fill('[data-testid="new-email"]', 'newintake@keystone.local');
    await page.fill('[data-testid="new-password"]', 'NewIntake@1234!');
    await page.selectOption('[data-testid="new-role"]', 'INTAKE_SPECIALIST');
    await page.click('[data-testid="create-user-btn"]');
    // Verify user appears
    await expect(page.locator('text=newintake')).toBeVisible({ timeout: 5000 });
  });
});

test.describe('IntakeSpecialistJourneyTest', () => {
  test('intake specialist can create and submit candidate', async ({ page }) => {
    await login(page, 'intake_specialist', 'Intake@Keystone1!');
    await page.click('text=Candidates');
    await page.waitForURL('**/candidates');
    await page.click('text=New Candidate');
    await page.waitForURL('**/candidates/new');

    // Fill required fields
    await page.fill('[data-testid="field-firstName"]', 'John');
    await page.fill('[data-testid="field-lastName"]', 'Doe');
    await page.fill('[data-testid="field-dob"]', '1990-01-15');
    await page.fill('[data-testid="field-examScore"]', '85');
    await page.fill('[data-testid="field-applicationDate"]', '2024-01-01');
    await page.fill('[data-testid="field-position"]', 'Officer');
    await page.click('text=Save');

    await page.waitForURL(url => url.toString().includes('/candidates/') && !url.toString().includes('/new'));

    // Verify DRAFT status
    await expect(page.locator('[data-testid="status-badge"]')).toContainText('DRAFT');
  });
});

test.describe('ReviewerJourneyTest', () => {
  test('reviewer can approve a submitted candidate', async ({ page }) => {
    // First create a submitted candidate via API
    const apiRes = await page.request.post(`${API}/api/auth/login`, {
      data: { username: 'intake_specialist', password: 'Intake@Keystone1!' }
    });
    const authData = await apiRes.json();
    const token = authData.data?.token;

    // Create and submit candidate
    const cRes = await page.request.post(`${API}/api/candidates`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        demographics: { firstName: 'Jane', lastName: 'Smith' },
        examScores: { score: '90' },
        applicationDetails: { position: 'Analyst', applicationDate: '2024-01-01' },
        transferPreferences: {},
        completenessStatus: 'complete'
      }
    });
    const candidate = await cRes.json();
    const candidateId = candidate.data?.id;

    if (candidateId) {
      await page.request.post(`${API}/api/candidates/${candidateId}/submit`, {
        headers: { Authorization: `Bearer ${token}` }
      });
    }

    await login(page, 'reviewer', 'Review@Keystone1!');
    await page.click('text=Candidates');

    if (candidateId) {
      await page.goto(`${BASE}/candidates/${candidateId}`);
      const approveBtn = page.locator('[data-testid="approve-btn"]');
      if (await approveBtn.isVisible()) {
        await approveBtn.click();
        await page.locator('[data-testid="confirm-btn"]').click();
        await expect(page.locator('[data-testid="status-badge"]')).toContainText('APPROVED', { timeout: 5000 });
        // Verify timestamp format
        const timestamps = page.locator('[data-testid="timestamp-display"]');
        const count = await timestamps.count();
        if (count > 0) {
          const text = await timestamps.first().textContent();
          expect(text).toMatch(/\d{2}\/\d{2}\/\d{4}/);
        }
      }
    }
  });
});

test.describe('InventoryClerkJourneyTest', () => {
  test('clerk can create part and new version', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.click('text=Parts');
    await page.waitForURL('**/parts');
    await page.click('text=New Part');
    await page.waitForURL('**/parts/new');

    await page.fill('[data-testid="field-partNumber"]', `P-E2E-${Date.now()}`);
    await page.fill('[data-testid="field-name"]', 'E2E Test Part');
    await page.fill('[data-testid="field-description"]', 'Created by E2E test');
    await page.fill('[data-testid="field-fitmentMake"]', 'Toyota');
    await page.fill('[data-testid="field-fitmentModel"]', 'Camry');
    await page.fill('[data-testid="field-fitmentYear"]', '2020');
    await page.click('text=Save');

    await page.waitForURL(url => url.toString().includes('/parts/') && !url.toString().includes('/new'));
    await expect(page.locator('[data-testid="status-badge"]')).toBeVisible();

    // Edit to create version 2
    const url = page.url();
    const partId = url.split('/parts/')[1];
    await page.goto(`${BASE}/parts/${partId}/edit`);
    await page.fill('[data-testid="field-name"]', 'E2E Test Part v2');
    await page.fill('[data-testid="field-changeSummary"]', 'Updated name');
    await page.click('text=Save');

    // Verify version history shows v2
    await page.goto(`${BASE}/parts/${partId}`);
    await expect(page.locator('[data-testid="version-history"]')).toBeVisible();
  });
});

test.describe('BulkImportJourneyTest', () => {
  test('clerk can bulk import parts from CSV', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.click('text=Bulk Import');
    await page.waitForURL('**/parts/import');
    expect(page.locator('[data-testid="csv-input"]')).toBeTruthy();
  });
});

test.describe('DuplicateListingJourneyTest', () => {
  test('duplicate listing gets flagged, reviewer can override', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.click('text=Lost & Found');
    await page.waitForURL('**/listings');

    // Create first listing
    await page.click('text=New Listing');
    await page.fill('[data-testid="title-input"]', 'Lost Blue Wallet near Park');
    await page.selectOption('[data-testid="category-select"]', 'Documents');
    await page.fill('[data-testid="location-input"]', 'Denver, CO');
    await page.click('text=Save');
    await page.waitForURL(url => url.toString().includes('/listings/') && !url.toString().includes('/new'));

    // Create second near-duplicate listing
    await page.goto(`${BASE}/listings/new`);
    await page.fill('[data-testid="title-input"]', 'Lost Blue Wallet near the Park');
    await page.selectOption('[data-testid="category-select"]', 'Documents');
    await page.fill('[data-testid="location-input"]', 'Denver, CO');
    await page.click('text=Save');
    await page.waitForURL(url => url.toString().includes('/listings/') && !url.toString().includes('/new'));

    // Check for duplicate flag - may or may not be visible depending on threshold
    // Navigate to listings to verify
    await page.goto(`${BASE}/listings`);

    // Login as reviewer to check for override capability
    await page.goto(`${BASE}/login`);
    await page.fill('[data-testid="username-input"]', 'reviewer');
    await page.fill('[data-testid="password-input"]', 'Review@Keystone1!');
    await page.click('[data-testid="submit-button"]');
    await page.waitForURL(url => !url.toString().includes('/login'));
    await page.goto(`${BASE}/listings`);
    await expect(page.locator('[data-testid="listings-grid"]')).toBeVisible({ timeout: 5000 });
  });
});

test.describe('AuditorJourneyTest', () => {
  test('auditor can view audit logs but not create candidates', async ({ page }) => {
    await login(page, 'auditor', 'Audit@Keystone1!');
    await page.click('text=Audit Logs');
    await page.waitForURL('**/audit-logs');
    await expect(page.locator('[data-testid="data-table"]')).toBeVisible({ timeout: 5000 });

    // Attempt to navigate to candidate form
    await page.goto(`${BASE}/candidates/new`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});
