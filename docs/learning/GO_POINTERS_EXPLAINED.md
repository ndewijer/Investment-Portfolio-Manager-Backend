# Go Pointers: Complete Guide

This document explains everything about pointers in Go, with a focus on why they're needed in `database/sql` operations.

## Table of Contents

1. [The Core Concept: Pass by Value](#the-core-concept-pass-by-value)
2. [The & and * Operators](#the--and--operators)
3. [Why Scan() Needs Pointers](#why-scan-needs-pointers)
4. [Common Mistakes](#common-mistakes)
5. [Complete Examples](#complete-examples)
6. [Quick Reference](#quick-reference)

---

## The Core Concept: Pass by Value

### Go Makes Copies

In Go, when you pass a variable to a function, **Go makes a copy** of that variable.

```go
func changeNumber(x int) {
    x = 100  // This changes the COPY, not the original
}

func main() {
    num := 5
    changeNumber(num)
    fmt.Println(num)  // Still prints 5, not 100!
}
```

**What happened:**
1. `num` has value `5`
2. `changeNumber` receives a **copy** of `5`
3. It changes the copy to `100`
4. The original `num` is still `5`

### The Solution: Pointers

To modify the original variable, pass its **memory address** (a pointer):

```go
func changeNumber(x *int) {  // x is a pointer to an int
    *x = 100  // Follow the pointer and change the value
}

func main() {
    num := 5
    changeNumber(&num)  // Pass the address of num
    fmt.Println(num)    // Prints 100!
}
```

---

## The & and * Operators

### The * Symbol Has TWO Different Meanings!

This is what confuses everyone learning Go.

#### Meaning 1: Part of the Type (Type Declaration)

```go
var ptr *string  // "*string" is a TYPE meaning "pointer to string"
var num *int     // "*int" is a TYPE meaning "pointer to int"

func DoSomething(p *Portfolio) {}  // Parameter type is "pointer to Portfolio"
```

Here `*` is **part of the type name**, not an operator. It declares that this variable will hold a pointer.

#### Meaning 2: Dereference Operator

```go
var name string = "John"
var ptr *string = &name  // ptr is a pointer to name

*ptr = "Jane"  // * means "follow the pointer and change what it points to"
fmt.Println(*ptr)  // * means "follow the pointer and get the value"
```

Here `*` is an **operator** that means "follow this pointer to get/set the value it points to."

### The & Symbol Has ONE Meaning: Address-of Operator

```go
var name string = "John"
var ptr *string = &name  // & means "get the memory address of name"

fmt.Printf("Value: %s\n", name)    // "John"
fmt.Printf("Address: %p\n", &name) // "0xc000010200" (example address)
fmt.Printf("Pointer: %p\n", ptr)   // "0xc000010200" (same address)
```

The `&` operator gives you the **memory address** where a variable lives.

---

## Visual Representation

### Memory Layout

```
┌─────────────────────────────────────┐
│  Your Code's Memory Space           │
├─────────────────────────────────────┤
│                                     │
│  Address: 0x1000                    │
│  Variable: name                     │
│  Value: "John"                      │
│                                     │
│  Address: 0x2000                    │
│  Variable: ptr                      │
│  Value: 0x1000  (points to name)    │
│                                     │
└─────────────────────────────────────┘
```

### Using & and * Together

```go
name := "John"
ptr := &name     // ptr = 0x1000 (address of name)
*ptr = "Jane"    // Follow 0x1000 and change value to "Jane"
```

```
Step 1: Create variable              Step 2: Get address
┌──────────────┐                    ┌──────────────┐     ┌──────────────┐
│ name: "John" │                    │ name: "John" │     │ ptr: 0x1000  │
│ (at 0x1000)  │                    │ (at 0x1000)  │     │              │
└──────────────┘                    └──────────────┘     └──────────────┘
                                           ↑                     │
                                           └─────────────────────┘
                                                 ptr points to name

Step 3: Dereference and modify
┌──────────────┐     ┌──────────────┐
│ name: "Jane" │ ←── │ ptr: 0x1000  │
│ (at 0x1000)  │     │   *ptr = ... │
└──────────────┘     └──────────────┘
   Changed!
```

---

## Why Scan() Needs Pointers

### The Problem Without Pointers

If `rows.Scan()` accepted values instead of pointers:

```go
var name string
var age int

// Hypothetical (doesn't work):
rows.Scan(name, age)
```

**What would happen:**

```
┌──────────────────────┐         ┌──────────────────────┐
│  Your Code           │         │  Scan() Function     │
├──────────────────────┤         ├──────────────────────┤
│  name = ""  (0x1000) │ ──Copy→ │  name_copy = ""      │
│  age  = 0   (0x2000) │ ──Copy→ │  age_copy  = 0       │
└──────────────────────┘         └──────────────────────┘
                                          │
                                          │ Scan reads DB
                                          ↓
                                  name_copy = "John"
                                  age_copy  = 30
                                          │
                                          │ Function returns
                                          ↓
                                  Copies are destroyed!

Your variables: name = "", age = 0  (still empty!)
```

The data would be **lost** when the function returns because it only modified the copies.

### The Solution: Pass Pointers

```go
var name string
var age int

rows.Scan(&name, &age)  // ✅ Pass addresses (pointers)
```

**What happens:**

```
┌──────────────────────┐         ┌──────────────────────┐
│  Your Code           │         │  Scan() Function     │
├──────────────────────┤         ├──────────────────────┤
│  name = ""  (0x1000) │ ──Addr→ │  ptr1 = 0x1000       │
│  age  = 0   (0x2000) │ ──Addr→ │  ptr2 = 0x2000       │
└──────────────────────┘         └──────────────────────┘
         ↑                                │
         │                                │ Scan reads DB
         │                                ↓
         │                        *ptr1 = "John"  (writes to 0x1000)
         │                        *ptr2 = 30      (writes to 0x2000)
         │                                │
         │                                │ Function returns
         │←───────────────────────────────┘
         │
    Your variables now have the values!
    name = "John"
    age  = 30
```

Scan receives the **addresses**, so it can **write directly to your original variables**.

---

## Why Can't You Use *name Instead of &name?

### The Critical Difference

```go
var name string = "John"

// ✅ This works - get the address:
ptr := &name
rows.Scan(&name)

// ❌ This doesn't work - can't dereference a non-pointer:
rows.Scan(*name)  // ERROR!
```

### Why *name Doesn't Work

**You can only use `*` on pointers!**

```go
var name string = "John"

// To use *, you need something that IS a pointer:
var ptr *string = &name  // ptr is a pointer to name
fmt.Println(*ptr)        // ✅ Works! Prints "John"

// But name is NOT a pointer, it's a string:
fmt.Println(*name)       // ❌ ERROR! "invalid indirect of name (type string)"
```

**Translation:**
- `*` means "follow this pointer"
- `name` is a string, not a pointer
- You can't "follow" something that isn't pointing anywhere
- The compiler prevents this mistake

### What Each Does

```go
var name string = "John"

&name   // ✅ "Give me the ADDRESS of name" → creates a pointer
*name   // ❌ ERROR! "Follow name as a pointer" → but name isn't a pointer!
```

Only after you have a pointer can you dereference it:

```go
var name string = "John"
var ptr *string = &name  // ptr is now a pointer

*ptr = "Jane"  // ✅ Follow ptr and change the value
```

---

## Complete Example: From Start to Finish

```go
package main

import "fmt"

func main() {
    // 1. Create a regular variable
    name := "John"
    fmt.Printf("Initial value: %s\n", name)

    // 2. Get its address (create a pointer)
    ptr := &name  // & = "address-of operator"
    fmt.Printf("Address of name: %p\n", &name)
    fmt.Printf("Pointer value: %p\n", ptr)  // Same address

    // 3. Dereference the pointer to read
    fmt.Printf("Value via pointer: %s\n", *ptr)  // * = "dereference operator"

    // 4. Dereference the pointer to write
    *ptr = "Jane"  // Follows ptr and modifies what it points to

    // 5. Check original variable
    fmt.Printf("After modification: %s\n", name)  // "Jane"

    // 6. What doesn't work:
    // fmt.Println(*name)  // ❌ ERROR! name is not a pointer

    // 7. Passing to a function
    modifyString(&name)  // Pass address
    fmt.Printf("After function: %s\n", name)  // "Modified"
}

func modifyString(s *string) {
    *s = "Modified"  // Dereference and assign
}
```

**Output:**
```
Initial value: John
Address of name: 0xc000010200
Pointer value: 0xc000010200
Value via pointer: John
After modification: Jane
After function: Modified
```

---

## In Your Portfolio Code

```go
for rows.Next() {
    var p Portfolio

    // Pass ADDRESSES so Scan can write to them
    err := rows.Scan(
        &p.ID,                  // "Here's where ID lives in memory"
        &p.Name,                // "Here's where Name lives in memory"
        &p.Description,         // "Here's where Description lives in memory"
        &p.IsArchived,          // "Here's where IsArchived lives in memory"
        &p.ExcludeFromOverview, // "Here's where ExcludeFromOverview lives in memory"
    )

    // Now Scan has written directly into these fields
    // p.ID = "abc-123"
    // p.Name = "My Portfolio"
    // etc.
}
```

**What Scan does internally (simplified):**

```go
func (rs *Rows) Scan(dest ...interface{}) error {
    // dest[0] is &p.ID (a pointer)

    for i, columnValue := range databaseRow {
        // Type assert to get the specific pointer type
        switch ptr := dest[i].(type) {
        case *string:
            *ptr = columnValue.String()  // Dereference and write
        case *int:
            *ptr = columnValue.Int()     // Dereference and write
        case *bool:
            *ptr = columnValue.Bool()    // Dereference and write
        }
    }

    return nil
}
```

Notice the `*ptr = value` - it **dereferences** the pointer to write to your original variable.

---

## Common Mistakes

### ❌ Mistake 1: Trying to Dereference a Non-Pointer

```go
var name string = "John"
fmt.Println(*name)  // ERROR! name is not a pointer
```

**Error:**
```
invalid indirect of name (type string)
```

**Fix:**
```go
var name string = "John"
var ptr *string = &name
fmt.Println(*ptr)  // ✅ Works!
```

### ❌ Mistake 2: Forgetting & When Passing to Scan

```go
var name string
rows.Scan(name)  // ERROR! Scan expects a pointer
```

**Error:**
```
cannot use name (type string) as type *interface {} in argument to rows.Scan
```

**Fix:**
```go
var name string
rows.Scan(&name)  // ✅ Pass the address
```

### ❌ Mistake 3: Using & on Something Already a Pointer

```go
var name string = "John"
var ptr *string = &name  // ptr is a pointer

rows.Scan(&ptr)  // Wrong! ptr is already a pointer
```

**Fix:**
```go
rows.Scan(ptr)  // ✅ Pass the pointer directly (no &)
```

### ❌ Mistake 4: Confusing Type Declaration with Dereference

```go
// Type declaration:
var ptr *string  // * is part of the type

// Dereference:
*ptr = "value"   // * is an operator
```

These look similar but are completely different!

---

## Why This Design?

Go chose "pass by value" by default for good reasons:

### 1. Safety
Can't accidentally modify variables you didn't mean to.

```go
func printName(name string) {
    name = "Changed"  // Only changes local copy
}

original := "John"
printName(original)
fmt.Println(original)  // Still "John"
```

### 2. Clarity
When you see `&`, you know "this function might change this variable."

```go
func modifyName(name *string) {  // Clear: might modify
    *name = "Changed"
}

original := "John"
modifyName(&original)  // & signals: "this might change"
```

### 3. Performance
For small values (int, bool), copying is faster than pointer indirection.

### 4. Explicitness
The `&` makes mutation obvious at the call site.

```go
updateUser(&user)   // ✅ Clear: user might be modified
updateUser(user)    // ✅ Clear: user won't be modified (copy)
```

---

## Other Common Go Functions That Need Pointers

### json.Unmarshal

```go
var user User
json.Unmarshal(data, &user)  // Needs to WRITE into user
```

### fmt.Scanf

```go
var name string
fmt.Scanf("%s", &name)  // Needs to WRITE into name
```

### sql.QueryRow.Scan

```go
var count int
db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)  // Needs to WRITE into count
```

### binary.Read

```go
var value int32
binary.Read(reader, binary.LittleEndian, &value)  // Needs to WRITE into value
```

**Pattern:** If a function needs to **modify** your variable, it requires a pointer.

---

## Quick Reference

### Operators

| Symbol | Context | Meaning | Example |
|--------|---------|---------|---------|
| `*` | Type declaration | "Pointer to" | `var ptr *string` |
| `*` | Operator | "Dereference" (follow pointer) | `*ptr = "value"` |
| `&` | Operator | "Address-of" (get pointer) | `ptr := &name` |

### Common Patterns

| Pattern | What It Does | When To Use |
|---------|--------------|-------------|
| `var p *Portfolio` | Declare pointer variable | When you need to store a pointer |
| `&portfolio` | Get address of variable | Passing to functions that need to modify |
| `*ptr` | Read value through pointer | Accessing what a pointer points to |
| `*ptr = value` | Write value through pointer | Modifying what a pointer points to |
| `rows.Scan(&name)` | Pass address to function | When function needs to write to your variable |

### Decision Tree: & or *?

```
Do you have a pointer?
├─ No, I have a regular variable
│  └─ Use & to get its address
│     Example: rows.Scan(&name)
│
└─ Yes, I have a pointer
   ├─ Want to pass it to a function?
   │  └─ Pass it directly (no & or *)
   │     Example: processPointer(ptr)
   │
   └─ Want to access/modify the value?
      └─ Use * to dereference
         Example: *ptr = "new value"
```

### Examples Side-by-Side

```go
// Creating variables
name := "John"           // Regular variable
ptr := &name            // Pointer variable (holds address)

// Reading values
fmt.Println(name)       // Direct access: "John"
fmt.Println(*ptr)       // Through pointer: "John"

// Writing values
name = "Jane"           // Direct write
*ptr = "Jane"           // Through pointer

// Passing to functions
processValue(name)      // Pass copy (can't modify original)
processPointer(&name)   // Pass address (can modify original)
processPointer(ptr)     // Pass existing pointer (can modify original)
```

---

## Summary

### The Core Rules

1. **Go passes everything by value** (makes copies)
2. **To modify a variable from a function, pass a pointer** (use `&`)
3. **The `&` operator gets a memory address** (creates a pointer)
4. **The `*` operator has two meanings:**
   - In types: "pointer to" (`var ptr *string`)
   - As operator: "dereference" (follow pointer: `*ptr`)
5. **You can only dereference pointers** (`*name` won't work if `name` isn't a pointer)

### Why Scan() Needs Pointers

```go
rows.Scan(&p.ID, &p.Name)  // ✅ Pass addresses
```

Because:
- Scan needs to **write** database values into your variables
- If you passed values, Scan would only modify copies
- By passing addresses (pointers), Scan can write to the original variables
- This is the only way to "return" multiple values through parameters in Go

### Remember

- **`&variable`** = "Give me the address" → Creates a pointer
- **`*pointer`** = "Follow the arrow" → Gets/sets the value
- **`*Type`** = "Pointer to Type" → Type declaration

When in doubt:
- Need to modify? Use `&` to pass the address
- Have a pointer? Use `*` to access the value
- Declaring a pointer type? Use `*` in the type

---

Happy coding! Once you understand pointers, you'll see this pattern everywhere in Go, and it will become second nature.
