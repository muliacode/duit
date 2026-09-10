<script lang="ts" setup>
import HelloWorld from './components/HelloWorld.vue';
import {useRoute} from "vue-router";
import {onMounted} from "vue";

const route = useRoute();
</script>

<template>
  <HelloWorld/>
</template>
