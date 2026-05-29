<script setup lang="ts">
import { NLayout, NLayoutContent, NLayoutSider, NMenu, type MenuOption } from "naive-ui";
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useAuthStore } from "@/stores/auth";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const menuOptions = computed<MenuOption[]>(() => [
	{
		label: "Inventory",
		key: "inventory",
	},
	{
		label: "Orders",
		key: "orders",
	},
	{
		label: "Deleted",
		key: "deleted",
	},
	{
		label: "My Profile",
		key: "profile",
	},
	{
		label: "Logout",
		key: "logout",
	},
]);

const selectedKey = computed(() => {
	if (route.path.startsWith("/app/profile")) {
		return "profile";
	}

	if (route.path.startsWith("/app/deleted")) {
		return "deleted";
	}

	if (route.path.startsWith("/app/orders")) {
		return "orders";
	}

	return "inventory";
});

async function handleSelect(key: string): Promise<void> {
	if (key === "logout") {
		await authStore.logout();
		await router.push("/login");
		return;
	}

	if (key === "profile") {
		await router.push("/app/profile");
		return;
	}

	if (key === "deleted") {
		await router.push("/app/deleted");
		return;
	}

	if (key === "orders") {
		await router.push("/app/orders");
		return;
	}

	await router.push("/app/inventory");
}
</script>

<template>
	<n-layout has-sider class="min-h-screen">
		<n-layout-sider :width="240" bordered>
			<div class="h-full bg-slate-950 p-4 text-slate-50">
				<div class="mb-8 px-1">
					<p class="text-xs tracking-widest text-slate-400 uppercase">Linktic</p>
					<h1 class="mt-1 text-lg font-semibold">Operations</h1>
				</div>
				<n-menu
					:options="menuOptions"
					:value="selectedKey"
					@update:value="handleSelect"
				/>
			</div>
		</n-layout-sider>

		<n-layout>
			<n-layout-content content-style="padding: 24px; min-height: 100svh;">
				<router-view />
			</n-layout-content>
		</n-layout>
	</n-layout>
</template>
