import AppLayoutView from "@views/AppLayoutView.vue";
import DeletedView from "@views/DeletedView.vue";
import InventoryView from "@views/InventoryView.vue";
import LoginView from "@views/LoginView.vue";
import OrdersView from "@views/OrdersView.vue";
import ProfileView from "@views/ProfileView.vue";
import { createRouter, createWebHistory } from "vue-router";

import { pinia } from "@/stores";
import { useAuthStore } from "@/stores/auth";

const router = createRouter({
	history: createWebHistory(),
	routes: [
		{
			path: "/",
			redirect: "/app",
		},
		{
			path: "/login",
			name: "login",
			component: LoginView,
		},
		{
			path: "/app",
			component: AppLayoutView,
			children: [
				{
					path: "",
					redirect: "/app/inventory",
				},
				{
					path: "inventory",
					name: "inventory",
					component: InventoryView,
				},
				{
					path: "deleted",
					name: "deleted",
					component: DeletedView,
				},
				{
					path: "orders",
					name: "orders",
					component: OrdersView,
				},
				{
					path: "profile",
					name: "profile",
					component: ProfileView,
				},
			],
		},
	],
});

router.beforeEach(async (to) => {
	const authStore = useAuthStore(pinia);

	if (authStore.status === "unknown") {
		await authStore.restoreSession();
	}

	if (to.path === "/login" && authStore.isAuthenticated) {
		return "/app/inventory";
	}

	if (to.path !== "/login" && !authStore.isAuthenticated) {
		return "/login";
	}

	return true;
});

export default router;