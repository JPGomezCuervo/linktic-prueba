import { useApi } from "@composables/useApi";

interface SignupPayload {
	name: string;
	email: string;
	password: string;
}

interface LoginPayload {
	email: string;
	password: string;
}

interface UpdateMePayload {
	name?: string;
	email?: string;
	password?: string;
	currentPassword?: string;
}

export interface AuthProfile {
	id: string;
	email: string;
	name: string;
	deleted: boolean;
	createdAt: number;
	updatedAt: number;
}

export function useAuth() {
	const { request } = useApi();

	async function signup(payload: SignupPayload): Promise<void> {
		await request("/api/auth/signup", {
			method: "POST",
			body: payload,
		});
	}

	async function login(payload: LoginPayload): Promise<void> {
		await request("/api/auth/login", {
			method: "POST",
			body: payload,
		});
	}

	async function me(): Promise<AuthProfile> {
		return request<AuthProfile>("/api/auth/me", {
			method: "GET",
		});
	}

	async function logout(): Promise<void> {
		await request("/api/auth/logout", {
			method: "POST",
		});
	}

	async function updateMe(payload: UpdateMePayload): Promise<AuthProfile> {
		return request<AuthProfile>("/api/auth/me", {
			method: "PATCH",
			body: payload,
		});
	}

	return {
		signup,
		login,
		me,
		logout,
		updateMe,
	};
}