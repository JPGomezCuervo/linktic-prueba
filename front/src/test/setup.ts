import { setActivePinia } from "pinia";
import { createTestingPinia } from "@pinia/testing";

export function setupTestPinia(options: Parameters<typeof createTestingPinia>[0] = {}) {
	const pinia = createTestingPinia({
		stubActions: false,
		...options,
	});
	setActivePinia(pinia);
	return pinia;
}

let fetchMock: ReturnType<typeof vi.fn> | null = null;

export function mockFetch(impl: (input: string, init?: RequestInit) => Promise<Response>) {
	if (fetchMock) {
		fetchMock.mockImplementation(impl);
	} else {
		fetchMock = vi.fn(impl);
		vi.stubGlobal("fetch", fetchMock);
	}
	return fetchMock;
}

export function mockFetchResponse(status: number, body: unknown, headers: Record<string, string> = {}) {
	return new Response(JSON.stringify(body), {
		status,
		headers: { "Content-Type": "application/json", ...headers },
	});
}

export function mockFetchError(status: number, message: string) {
	return new Response(JSON.stringify({ message }), { status });
}

export function resetFetchMock() {
	if (fetchMock) {
		fetchMock.mockReset();
	}
}

beforeEach(() => {
	vi.useFakeTimers();
});

afterEach(() => {
	vi.useRealTimers();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
	fetchMock = null;
});
