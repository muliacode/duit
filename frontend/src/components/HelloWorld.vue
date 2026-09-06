<script lang="ts" setup>
import {reactive} from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {Greet} from '../../wailsjs/go/main/App'

const data = reactive({
  name: "",
  resultText: "Please enter your name below 👇",
})

function greet() {
  Greet(data.name).then(result => {
    data.resultText = result
  })
}

</script>

<template>
  <main>
    <div class="flex w-full max-w-sm items-end space-x-2 m-20">
      <div class="grid w-full max-w-sm items-center gap-1.5">
        <Label id="result" for="result">{{ data.resultText }}</Label>
        <Input id="name" type="text" v-model="data.name" placeholder="John Doe" />
      </div>
      <Button @click="greet">Greet</Button>
    </div>
  </main>
</template>
