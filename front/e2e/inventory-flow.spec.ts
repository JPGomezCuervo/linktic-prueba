import { test, expect } from "@playwright/test";

const TEST_EMAIL = `e2e-${Date.now()}@test.com`;
const TEST_PASSWORD = "e2etestpass123";
const TEST_NAME = "E2E Test User";
const PRODUCT_NAME = "E2E Test Widget";
const UPDATED_PRODUCT_NAME = "E2E Updated Widget";

test("full inventory flow: signup, login, create, edit, delete, verify deleted", async ({ page }) => {
	await test.step("signup", async () => {
		await page.goto("/login");
		await expect(page).toHaveURL(/\/login/);

		await page.getByRole("button", { name: "Signup" }).click();
		await expect(page.getByPlaceholder("Jane Doe")).toBeVisible();

		await page.getByPlaceholder("Jane Doe").fill(TEST_NAME);
		await page.getByPlaceholder("you@email.com").fill(TEST_EMAIL);
		await page.getByPlaceholder("At least 8 characters").fill(TEST_PASSWORD);
		await page.getByRole("button", { name: "Create account" }).click();

		await expect(page.getByText("Account created")).toBeVisible();
	});

	await test.step("login", async () => {
		await page.getByPlaceholder("you@company.com").fill(TEST_EMAIL);
		await page.getByPlaceholder("********").fill(TEST_PASSWORD);
		await page.getByRole("button", { name: "Login" }).click();

		await expect(page).toHaveURL(/\/app\/inventory/);
	});

	await test.step("create product", async () => {
		await page.getByRole("button", { name: "Create product" }).click();
		await expect(page.getByPlaceholder("Product name")).toBeVisible();

		await page.getByPlaceholder("Product name").fill(PRODUCT_NAME);
		const createModal = page.locator(".n-modal:visible");
		await createModal.locator(".n-input-number input").first().fill("50");
		await createModal.locator(".n-input-number input").nth(1).fill("19.99");

		await createModal.getByRole("button", { name: "Create", exact: true }).click();
		await expect(page.getByText("Item created successfully")).toBeVisible();
		await expect(page.getByText(PRODUCT_NAME).first()).toBeVisible();
	});

	await test.step("edit product name", async () => {
		const row = page.locator("tr", { hasText: PRODUCT_NAME }).first();
		await row.getByRole("button", { name: "Actions" }).click();
		await page.getByText("Edit", { exact: true }).click();
		const editModal = page.locator(".n-modal:visible");
		await editModal.getByPlaceholder("Product name").fill(UPDATED_PRODUCT_NAME);
		await page.getByRole("button", { name: "Save changes" }).click();

		await expect(page.getByText("Item updated successfully")).toBeVisible();
		await expect(page.getByText(UPDATED_PRODUCT_NAME).first()).toBeVisible();
	});

	await test.step("delete product", async () => {
		const row = page.locator("tr", { hasText: UPDATED_PRODUCT_NAME }).first();
		await row.getByRole("button", { name: "Actions" }).click();
		await page.getByText("Delete", { exact: true }).click();
		await expect(page.getByRole("button", { name: "Delete" })).toBeVisible();
		await page.getByRole("button", { name: "Delete" }).click();

		await expect(page.getByText("Item deleted successfully")).toBeVisible();
	});

	await test.step("verify in deleted items", async () => {
		await page.locator(".n-layout-sider").getByText("Deleted").click();
		await expect(page).toHaveURL(/\/app\/deleted/);
		await expect(page.getByText(UPDATED_PRODUCT_NAME).first()).toBeVisible();
	});
});
