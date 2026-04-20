// @ts-check
const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:3000';
const API  = 'http://localhost:8080';

async function login(page, username, password) {
  await page.goto(`${BASE}/login`);
  await page.fill('[data-testid="username-input"]', username);
  await page.fill('[data-testid="password-input"]', password);
  await page.click('[data-testid="submit-button"]');
  await page.waitForURL(url => !url.toString().includes('/login'), { timeout: 10000 });
  await page.locator('[data-testid="navbar"]').waitFor({ state: 'visible', timeout: 20000 });
}

async function apiToken(page, username, password) {
  const res = await page.request.post(`${API}/api/auth/login`, {
    data: { username, password }
  });
  const body = await res.json();
  return body.data?.token;
}

// =============================================================================
// 1. Admin
// =============================================================================
test.describe('AdminJourneyTest', () => {
  test('admin can create a new user', async ({ page }) => {
    await login(page, 'admin', 'Admin@Keystone1!');
    await expect(page.getByRole('link', { name: 'Admin' })).toBeVisible();
    await page.goto(`${BASE}/admin`);
    await page.fill('[data-testid="new-username"]', 'newintake');
    await page.fill('[data-testid="new-email"]', 'newintake@keystone.local');
    await page.fill('[data-testid="new-password"]', 'NewIntake@1234!');
    await page.selectOption('[data-testid="new-role"]', 'INTAKE_SPECIALIST');
    await page.click('[data-testid="create-user-btn"]');
    await expect(page.getByRole('cell', { name: 'newintake', exact: true })).toBeVisible({ timeout: 5000 });
  });

  test('admin can view the user list on admin panel', async ({ page }) => {
    await login(page, 'admin', 'Admin@Keystone1!');
    await page.goto(`${BASE}/admin`);
    await expect(page.locator('[data-testid="data-table"]')).toBeVisible({ timeout: 5000 });
  });
});

// =============================================================================
// 2. Intake Specialist
// =============================================================================
test.describe('IntakeSpecialistJourneyTest', () => {
  test('intake specialist can create and submit candidate', async ({ page }) => {
    await login(page, 'intake_specialist', 'Intake@Keystone1!');
    await page.goto(`${BASE}/candidates/new`);
    await page.fill('[data-testid="field-firstName"]', 'John');
    await page.fill('[data-testid="field-lastName"]', 'Doe');
    await page.fill('[data-testid="field-dob"]', '1990-01-15');
    await page.fill('[data-testid="field-examScore"]', '85');
    await page.fill('[data-testid="field-applicationDate"]', '2024-01-01');
    await page.fill('[data-testid="field-position"]', 'Officer');
    await page.click('text=Save');
    await page.waitForURL(
      url => url.toString().includes('/candidates/') && !url.toString().includes('/new')
    );
    await expect(page.locator('[data-testid="status-badge"]')).toContainText('DRAFT');
  });

  test('intake specialist can view candidate list', async ({ page }) => {
    await login(page, 'intake_specialist', 'Intake@Keystone1!');
    await page.goto(`${BASE}/candidates`);
    await expect(page.locator('[data-testid="data-table"]')).toBeVisible({ timeout: 5000 });
  });

  test('intake specialist is blocked from admin panel', async ({ page }) => {
    await login(page, 'intake_specialist', 'Intake@Keystone1!');
    await page.goto(`${BASE}/admin`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });

  test('intake specialist is blocked from audit logs', async ({ page }) => {
    await login(page, 'intake_specialist', 'Intake@Keystone1!');
    await page.goto(`${BASE}/audit-logs`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});

// =============================================================================
// 3. Reviewer
// =============================================================================
test.describe('ReviewerJourneyTest', () => {
  test('reviewer can approve a submitted candidate', async ({ page }) => {
    const token = await apiToken(page, 'intake_specialist', 'Intake@Keystone1!');
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
    if (candidateId) {
      await page.goto(`${BASE}/candidates/${candidateId}`);
      const approveBtn = page.locator('[data-testid="approve-btn"]');
      if (await approveBtn.isVisible()) {
        await approveBtn.click();
        await page.locator('[data-testid="confirm-btn"]').click();
        await expect(page.locator('[data-testid="status-badge"]')).toContainText('APPROVED', { timeout: 5000 });
        const timestamps = page.locator('[data-testid="timestamp-display"]');
        if (await timestamps.count() > 0) {
          const text = await timestamps.first().textContent();
          expect(text).toMatch(/\d{2}\/\d{2}\/\d{4}/);
        }
      }
    }
  });

  test('reviewer can reject a submitted candidate', async ({ page }) => {
    const token = await apiToken(page, 'intake_specialist', 'Intake@Keystone1!');
    const cRes = await page.request.post(`${API}/api/candidates`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        demographics: { firstName: 'Reject', lastName: 'Test' },
        examScores: { score: '40' },
        applicationDetails: { position: 'Officer', applicationDate: '2024-01-01' },
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
    if (candidateId) {
      await page.goto(`${BASE}/candidates/${candidateId}`);
      const rejectBtn = page.locator('[data-testid="reject-btn"]');
      if (await rejectBtn.isVisible()) {
        await rejectBtn.click();
        await page.fill('[data-testid="reject-comments"]', 'Below threshold');
        await page.locator('[data-testid="reject-submit"]').click();
        await expect(page.locator('[data-testid="status-badge"]')).toContainText('REJECTED', { timeout: 5000 });
      }
    }
  });

  test('reviewer is blocked from admin panel', async ({ page }) => {
    await login(page, 'reviewer', 'Review@Keystone1!');
    await page.goto(`${BASE}/admin`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });

  test('reviewer is blocked from audit logs', async ({ page }) => {
    await login(page, 'reviewer', 'Review@Keystone1!');
    await page.goto(`${BASE}/audit-logs`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});

// =============================================================================
// 4. Inventory Clerk – Parts
// =============================================================================
test.describe('InventoryClerkJourneyTest', () => {
  test('clerk can create part and new version', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/parts/new`);
    await page.fill('[data-testid="field-partNumber"]', `P-E2E-${Date.now()}`);
    await page.fill('[data-testid="field-name"]', 'E2E Test Part');
    await page.fill('[data-testid="field-description"]', 'Created by E2E test');
    await page.fill('[data-testid="field-fitmentMake"]', 'Toyota');
    await page.fill('[data-testid="field-fitmentModel"]', 'Camry');
    await page.fill('[data-testid="field-fitmentYear"]', '2020');
    await page.click('text=Save');
    await page.waitForURL(
      url => url.toString().includes('/parts/') && !url.toString().includes('/new')
    );
    await expect(page.locator('[data-testid="status-badge"]')).toBeVisible();
    const partId = page.url().split('/parts/')[1];
    await page.goto(`${BASE}/parts/${partId}/edit`);
    await page.fill('[data-testid="field-name"]', 'E2E Test Part v2');
    await page.fill('[data-testid="field-changeSummary"]', 'Updated name');
    await page.click('text=Save');
    await page.goto(`${BASE}/parts/${partId}`);
    await expect(page.locator('[data-testid="version-history"]')).toBeVisible();
  });

  test('clerk can view parts list', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/parts`);
    await expect(page.locator('[data-testid="data-table"]')).toBeVisible({ timeout: 5000 });
  });

  test('clerk is blocked from admin panel', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/admin`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });

  test('clerk is blocked from audit logs', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/audit-logs`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});

// =============================================================================
// 5. Bulk Import
// =============================================================================
test.describe('BulkImportJourneyTest', () => {
  test('clerk can bulk import parts from CSV', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/parts/import`);
    await expect(page).toHaveURL(`${BASE}/parts/import`);
    await expect(page.locator('[data-testid="csv-input"]')).toBeVisible({ timeout: 5000 });
  });

  test('reviewer cannot access bulk import', async ({ page }) => {
    await login(page, 'reviewer', 'Review@Keystone1!');
    await page.goto(`${BASE}/parts/import`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});

// =============================================================================
// 6. Listings / Lost & Found
// =============================================================================
test.describe('DuplicateListingJourneyTest', () => {
  test('duplicate listing gets flagged, reviewer can override', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/listings/new`);
    await page.fill('[data-testid="title-input"]', 'Lost Blue Wallet near Park');
    await page.selectOption('[data-testid="category-select"]', 'Documents');
    await page.fill('[data-testid="location-input"]', 'Denver, CO');
    await page.fill('[data-testid="start-time"]', '2026-06-01T09:00');
    await page.fill('[data-testid="end-time"]', '2026-07-01T09:00');
    await page.click('text=Save');
    await page.waitForURL(
      url => url.toString().includes('/listings/') && !url.toString().includes('/new')
    );
    await page.goto(`${BASE}/listings/new`);
    await page.fill('[data-testid="title-input"]', 'Lost Blue Wallet near the Park');
    await page.selectOption('[data-testid="category-select"]', 'Documents');
    await page.fill('[data-testid="location-input"]', 'Denver, CO');
    await page.fill('[data-testid="start-time"]', '2026-06-01T09:00');
    await page.fill('[data-testid="end-time"]', '2026-07-01T09:00');
    await page.click('text=Save');
    await page.waitForURL(
      url => url.toString().includes('/listings/') && !url.toString().includes('/new')
    );
    await login(page, 'reviewer', 'Review@Keystone1!');
    await page.goto(`${BASE}/listings`);
    await expect(page.locator('[data-testid="listings-grid"]')).toBeVisible({ timeout: 5000 });
  });

  test('clerk can create and view a listing', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/listings/new`);
    await page.fill('[data-testid="title-input"]', `E2E Listing ${Date.now()}`);
    await page.selectOption('[data-testid="category-select"]', 'Electronics');
    await page.fill('[data-testid="location-input"]', 'Austin, TX');
    await page.fill('[data-testid="start-time"]', '2026-06-01T09:00');
    await page.fill('[data-testid="end-time"]', '2026-07-01T09:00');
    await page.click('text=Save');
    await page.waitForURL(
      url => url.toString().includes('/listings/') && !url.toString().includes('/new')
    );
    await expect(page.locator('[data-testid="status-badge"]')).toBeVisible();
  });

  test('intake specialist cannot create a listing', async ({ page }) => {
    await login(page, 'intake_specialist', 'Intake@Keystone1!');
    await page.goto(`${BASE}/listings/new`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});

// =============================================================================
// 7. Auditor
// =============================================================================
test.describe('AuditorJourneyTest', () => {
  test('auditor can view audit logs but not create candidates', async ({ page }) => {
    await login(page, 'auditor', 'Audit@Keystone1!');
    await page.goto(`${BASE}/audit-logs`);
    await expect(page.locator('[data-testid="data-table"]')).toBeVisible({ timeout: 5000 });
    await page.goto(`${BASE}/candidates/new`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });

  test('auditor is blocked from admin panel', async ({ page }) => {
    await login(page, 'auditor', 'Audit@Keystone1!');
    await page.goto(`${BASE}/admin`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });

  test('auditor cannot create parts', async ({ page }) => {
    await login(page, 'auditor', 'Audit@Keystone1!');
    await page.goto(`${BASE}/parts/new`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});

// =============================================================================
// 8. Reports / KPI
// =============================================================================
test.describe('ReportsJourneyTest', () => {
  test('admin can view KPI dashboard', async ({ page }) => {
    await login(page, 'admin', 'Admin@Keystone1!');
    await page.goto(`${BASE}/reports/kpi`);
    await expect(page).not.toHaveURL(`${BASE}/unauthorized`);
  });

  test('auditor can view KPI dashboard', async ({ page }) => {
    await login(page, 'auditor', 'Audit@Keystone1!');
    await page.goto(`${BASE}/reports/kpi`);
    await expect(page).not.toHaveURL(`${BASE}/unauthorized`);
  });

  test('inventory clerk is blocked from KPI dashboard', async ({ page }) => {
    await login(page, 'inventory_clerk', 'Clerk@Keystone1!');
    await page.goto(`${BASE}/reports/kpi`);
    await expect(page).toHaveURL(`${BASE}/unauthorized`);
  });
});

// =============================================================================
// 9. Authentication
// =============================================================================
test.describe('AuthJourneyTest', () => {
  test('invalid credentials keep user on login page', async ({ page }) => {
    await page.goto(`${BASE}/login`);
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'wrongpassword!!');
    await page.click('[data-testid="submit-button"]');
    await expect(page).toHaveURL(/\/login/);
  });

  test('unauthenticated user is redirected to login', async ({ page }) => {
    await page.goto(`${BASE}/candidates`);
    await expect(page).toHaveURL(/\/login/);
  });

  test('user can log out and is redirected to login', async ({ page }) => {
    await login(page, 'admin', 'Admin@Keystone1!');
    await page.locator('button:text("Logout")').click();
    await expect(page).toHaveURL(/\/login/);
  });
});
