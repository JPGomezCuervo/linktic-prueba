import { describe, it, expect, vi, beforeEach } from "vitest";
import { setActivePinia } from "pinia";
import { createTestingPinia } from "@pinia/testing";

import { useAuthStore } from "@/stores/auth";
import { mockFetch, mockFetchResponse, mockFetchError, resetFetchMock } from "@test/setup";

const testProfile = {
	id: "user-1",
	email: "test@example.com",
	name: "Test User",
	deleted: false,
	createdAt: 1000,
	updatedAt: 1000,
};

describe("useAuthStore", () => {
	beforeEach(() => {
		setActivePinia(
			createTestingPinia({
				stubActions: false,
			}),
		);
		resetFetchMock();
	});

	it("initializes with unknown status and null profile", () => {
		const store = useAuthStore();
		expect(store.status).toBe("unknown");
		expect(store.profile).toBeNull();
		expect(store.isAuthenticated).toBe(false);
	});

	it("restoreSession sets authenticated on success", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/me") {
				return Promise.resolve(mockFetchResponse(200, testProfile));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		await store.restoreSession();

		expect(store.status).toBe("authenticated");
		expect(store.profile).toEqual(testProfile);
		expect(store.isAuthenticated).toBe(true);
	});

	it("restoreSession sets guest on 401", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/me") {
				return Promise.resolve(mockFetchError(401, "Unauthorized"));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		await store.restoreSession();

		expect(store.status).toBe("guest");
		expect(store.profile).toBeNull();
		expect(store.isAuthenticated).toBe(false);
	});

	it("restoreSession sets guest on network error", async () => {
		mockFetch(() => Promise.reject(new Error("Network error")));

		const store = useAuthStore();
		await store.restoreSession();

		expect(store.status).toBe("guest");
		expect(store.profile).toBeNull();
	});

	it("restoreSession is idempotent when already authenticated", async () => {
		const meSpy = vi.fn();
		mockFetch((url) => {
			if (url === "/api/auth/me") {
				meSpy();
				return Promise.resolve(mockFetchResponse(200, testProfile));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		store.status = "authenticated";
		store.profile = testProfile;

		await store.restoreSession();

		expect(meSpy).not.toHaveBeenCalled();
		expect(store.profile).toEqual(testProfile);
	});

	it("login sets authenticated on success", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/login") {
				return Promise.resolve(mockFetchResponse(200, {}));
			}
			if (url === "/api/auth/me") {
				return Promise.resolve(mockFetchResponse(200, testProfile));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		await store.login("test@example.com", "password123");

		expect(store.status).toBe("authenticated");
		expect(store.profile).toEqual(testProfile);
	});

	it("login throws on failure", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/login") {
				return Promise.resolve(mockFetchError(401, "Invalid credentials"));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		await expect(store.login("test@example.com", "wrong")).rejects.toThrow();
		expect(store.status).toBe("unknown");
	});

	it("signup calls api without side effects on success", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/signup") {
				return Promise.resolve(mockFetchResponse(201, {}));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		await store.signup("Test", "test@example.com", "password123");

		expect(store.status).toBe("unknown");
		expect(store.profile).toBeNull();
	});

	it("signup throws on failure", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/signup") {
				return Promise.resolve(mockFetchError(409, "Email already exists"));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		await expect(store.signup("Test", "test@example.com", "password123")).rejects.toThrow();
	});

	it("logout clears profile and sets guest", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/logout") {
				return Promise.resolve(mockFetchResponse(200, {}));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		store.status = "authenticated";
		store.profile = testProfile;

		await store.logout();

		expect(store.status).toBe("guest");
		expect(store.profile).toBeNull();
		expect(store.isAuthenticated).toBe(false);
	});

	it("updateProfile updates profile on success", async () => {
		const updatedProfile = { ...testProfile, name: "Updated Name" };

		mockFetch((url) => {
			if (url === "/api/auth/me") {
				return Promise.resolve(mockFetchResponse(200, updatedProfile));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		store.status = "authenticated";
		store.profile = testProfile;

		await store.updateProfile({ name: "Updated Name" });

		expect(store.profile).toEqual(updatedProfile);
		expect(store.profile?.name).toBe("Updated Name");
	});

	it("updateProfile throws on failure and preserves existing profile", async () => {
		mockFetch((url) => {
			if (url === "/api/auth/me") {
				return Promise.resolve(mockFetchError(400, "Invalid email"));
			}
			return Promise.reject(new Error("Not found"));
		});

		const store = useAuthStore();
		store.status = "authenticated";
		store.profile = testProfile;

		await expect(store.updateProfile({ email: "invalid" })).rejects.toThrow();
		expect(store.profile).toEqual(testProfile);
	});
});
