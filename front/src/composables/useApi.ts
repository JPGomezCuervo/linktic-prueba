export interface ApiError {
	status: number;
	message: string;
}

interface RequestOptions {
	method?: "GET" | "POST" | "PATCH" | "DELETE";
	body?: unknown;
	headers?: Record<string, string>;
}

export function useApi() {
	async function request<T>(url: string, options: RequestOptions = {}): Promise<T> {
		const response = await fetch(url, {
			method: options.method ?? "GET",
			credentials: "include",
			headers: {
				"Content-Type": "application/json",
				...options.headers,
			},
			body: options.body === undefined ? undefined : JSON.stringify(options.body),
		});

		if (!response.ok) {
			let message = response.statusText;
			const rawBody = await response.text();
			if (rawBody.trim() !== "") {
				try {
					const payload = JSON.parse(rawBody) as { message?: string };
					message = payload.message ?? rawBody;
				} catch {
					message = rawBody;
				}
			}

			throw {
				status: response.status,
				message,
			} satisfies ApiError;
		}

		if (response.status === 204) {
			return undefined as T;
		}

		const data = await response.json();
		if (data === null || data === undefined) {
			throw {
				status: response.status,
				message: "Server returned an empty response.",
			} satisfies ApiError;
		}
		if (
			typeof data === "object" &&
			"items" in data &&
			data.items === null
		) {
			data.items = [];
		}
		return data as T;
	}

	return {
		request,
	};
}